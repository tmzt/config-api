package config

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"

	"gorm.io/gorm"

	"github.com/tmzt/config-api/util"
)

type ConfigContextService struct {
	logger     util.SetRequestLogger
	db         *gorm.DB
	rdb        *redis.Client
	refService *ConfigReferenceService
	// versionService *configVersionService
	cacheService *util.CacheService
}

func NewConfigContextService(db *gorm.DB, rdb *redis.Client, cacheService *util.CacheService) *ConfigContextService {
	logger := util.NewLogger("ConfigContextService", 0)

	refService := NewConfigReferenceService(db, rdb, cacheService)
	// versionService := newConfigVersionService(db, rdb, cacheService, refService)

	return &ConfigContextService{
		logger:     logger,
		db:         db,
		rdb:        rdb,
		refService: refService,
		// versionService: versionService,
	}
}

func (s *ConfigContextService) CreateHandle(scope util.ScopeKind, accountId util.AccountId, userId util.UserId) ConfigContextHandle {
	// configCtx := &configContext{
	// 	accountId: accountId,
	// 	userId:    userId,
	// }

	w := &configContextWrapper{
		scope:     scope,
		accountId: accountId,
		userId:    userId,
		// configContext: configCtx,
	}

	internalHandle := &configContextHandleInternal{
		scope:                scope,
		accountId:            accountId,
		userId:               userId,
		configContextWrapper: w,
	}

	rawHandle := interface{}(internalHandle)

	handle, ok := rawHandle.(ConfigContextHandle)
	if !ok {
		s.logger.Printf("********** CreateHandle: Error creating handle for account (cannot cast internal to ConfigContextHandle) %s and user %s: expected ConfigContextHandle, got %T\n", accountId, userId, rawHandle)
		return nil
	}

	return handle
}

func (s *ConfigContextService) LoadContext(ctx context.Context, tx *gorm.DB, handle ConfigContextHandle) error {
	w := s.getWrapperInternal(handle)
	if w == nil {
		s.logger.Printf("LoadContext: Invalid handle\n")
		return fmt.Errorf("invalid config context handle")
	}

	// if w.configContext == nil {
	// 	s.logger.Printf("LoadContext: Invalid handle, missing config context\n")
	// 	return fmt.Errorf("Invalid handle, missing config context")
	// }

	scope := w.scope
	accountId := w.accountId
	userId := w.userId

	var configCtx *configContext

	dirty := false

	cacheQuery := configContext{
		accountId: accountId,
		userId:    userId,
	}

	cachedCtx, ok, err := s.cacheService.GetCachedObject(ctx, cacheQuery)
	if err != nil {
		s.logger.Printf("loadContext: Error getting cached config context: %s\n", err)
	} else if ok {
		v, ok := cachedCtx.(*configContext)
		if !ok {
			s.logger.Printf("loadContext: Error getting cached config context: expected *configContext, got %T\n", cachedCtx)
		} else {
			configCtx = v
		}
	}

	if configCtx == nil {
		configCtx = &configContext{
			accountId: accountId,
			userId:    userId,
		}
		dirty = true
	}

	if configCtx.accountRoot == nil {
		// TODO: Consider whether we want to cache refs or just the context
		accountRoot, err := s.refService.GetRecord(ctx, tx, util.ScopeKindAccount, accountId, userId, ConfigReferenceKindRoot)
		if err != nil {
			s.logger.Printf("loadContext: Error getting account root for account %s: %v\n", accountId, err)
			return err
		} else if accountRoot != nil {
			configCtx.accountRoot = accountRoot.CurrentRef
		} else {
			// An empty account root was created, so add a note
			configCtx.accountRoot.Note = util.StrPtr(fmt.Sprintf("autocreated: account %s root", accountId))
			configCtx.accountRootIsDirty = true
		}
		dirty = true
	}

	if scope == util.ScopeKindUser && configCtx.userRoot == nil {
		userRoot, err := s.refService.GetRecord(ctx, tx, util.ScopeKindUser, accountId, userId, ConfigReferenceKindRoot)
		if err != nil {
			s.logger.Printf("loadContext: Error getting user root for account %s and user %s: %v\n", accountId, userId, err)
			return err
		} else if userRoot != nil {
			configCtx.userRoot = userRoot.CurrentRef
			configCtx.userRootIsDirty = true
			configCtx.userRootSkipDb = true
		} else {
			// An empty user root was created, so add a note
			configCtx.userRoot.Note = util.StrPtr(fmt.Sprintf("autocreated: account %s user %s root", accountId, userId))
			configCtx.userRootIsDirty = true
		}
		dirty = true
	}

	if configCtx.current == nil {
		current, err := s.refService.GetRecord(ctx, tx, util.ScopeKindAccount, accountId, userId, ConfigReferenceKindHead)
		if err != nil {
			s.logger.Printf("loadContext: Error getting current ref for account %s and user %s: %v\n", accountId, userId, err)
			return err
		} else if current != nil {
			configCtx.current = current.CurrentRef
			configCtx.parent = current.ParentRef
			configCtx.currentIsDirty = true
			configCtx.currentSkipDb = true
		}
		dirty = true
	}

	// if configCtx.dirty {
	// 	s.logger.Printf("loadContext: Saving context for account %s and user %s to in-memory cache\n", accountId, userId)

	// 	if err := s.saveContext(ctx, tx, accountId, userId, configCtx); err != nil {
	// 		s.logger.Printf("loadContext: Error saving config context for account %s and user %s: %v\n", accountId, userId, err)
	// 		return nil, err
	// 	}
	// }

	// w := &configContextWrapper{
	// 	configContext: configCtx,
	// }

	if dirty {
		_, err := s.save(ctx, tx, w, true)
		if err != nil {
			s.logger.Printf("loadContext: Error saving config context for account %s and user %s: %v\n", accountId, userId, err)
			return err
		}
	}

	// internalHandle := &configContextHandleInternal{
	// 	accountId:            accountId,
	// 	userId:               userId,
	// 	configContextWrapper: w,
	// }

	// rawHandle := interface{}(internalHandle)
	// handle, _ := rawHandle.(ConfigContextHandle)

	// return handle, nil
	return nil
}

type configContextWrapper struct {
	lock  sync.Mutex
	dirty atomic.Bool

	scope     util.ScopeKind
	accountId util.AccountId
	userId    util.UserId

	configContext *configContext
}

type configContext struct {
	scope     util.ScopeKind
	accountId util.AccountId
	userId    util.UserId

	accountRoot *ConfigVersionRef
	userRoot    *ConfigVersionRef
	current     *ConfigVersionRef
	parent      *ConfigVersionRef

	accountRootIsDirty bool
	userRootIsDirty    bool
	currentIsDirty     bool

	accountRootSkipDb bool
	userRootSkipDb    bool
	currentSkipDb     bool
}

type ConfigContextHandle interface {
	SetData(ctx context.Context, tx *gorm.DB, cdo *ConfigDataObject, data util.Data) error
}

// TODO: Remove
type ConfigSettingHandle ConfigContextHandle

type configContextHandleInternal struct {
	scope                util.ScopeKind
	accountId            util.AccountId
	userId               util.UserId
	configContextWrapper *configContextWrapper
}

func (h *configContextHandleInternal) SetData(ctx context.Context, tx *gorm.DB, cdo *ConfigDataObject, data util.Data) error {
	logger := util.NewLogger("configContextHandleInternal", 0)

	logger.Printf("***** SetData: Not implemented\n")
	return nil
}

func configContextCacheKey(scope util.ScopeKind, accountId util.AccountId, userId util.UserId) string {
	userIdStr := ""
	if scope == util.ScopeKindUser {
		userIdStr = string(userId)
	}
	return fmt.Sprintf("appsub:config_context:scope=%s:account_id=%s:user_id=%s", scope, accountId, userIdStr)
}

func (c configContext) CacheKey() string {
	return configContextCacheKey(c.scope, c.accountId, c.userId)
}

func (c configContext) Ttl() time.Duration {
	return 1 * time.Hour
}

func (s *ConfigContextService) getWrapperInternal(handle ConfigContextHandle) *configContextWrapper {
	if handle == nil {
		s.logger.Printf("getWrapperInternal: Invalid handle (nil)\n")
		return nil
	}

	rawHandle := handle.(interface{})

	if internalHandle, ok := rawHandle.(*configContextHandleInternal); ok {
		return internalHandle.configContextWrapper
	}

	return nil
}

func (s *ConfigContextService) GetAccountRoot(handle ConfigContextHandle) *ConfigVersionRef {
	w := s.getWrapperInternal(handle)
	if w == nil {
		s.logger.Printf("GetAccountRoot: Invalid handle\n")
		return nil
	}

	return w.configContext.accountRoot
}

func (s *ConfigContextService) SetAccountRoot(handle ConfigContextHandle, ref *ConfigVersionRef) bool {
	w := s.getWrapperInternal(handle)
	if w == nil {
		s.logger.Printf("SetAccountRoot: Invalid handle\n")
		return false
	}

	// Take the lock
	if !w.lock.TryLock() {
		return false
	}
	defer w.lock.Unlock()

	c := w.configContext

	c.accountRoot = ref
	c.accountRootIsDirty = true
	w.dirty.Store(true)

	return true
}

func (s *ConfigContextService) GetUserRoot(handle ConfigContextHandle) *ConfigVersionRef {
	w := s.getWrapperInternal(handle)
	if w == nil {
		s.logger.Printf("GetUserRoot: Invalid handle\n")
		return nil
	}

	return w.configContext.userRoot
}

func (s *ConfigContextService) SetUserRoot(handle ConfigContextHandle, ref *ConfigVersionRef) bool {
	w := s.getWrapperInternal(handle)
	if w == nil {
		s.logger.Printf("SetUserRoot: Invalid handle\n")
		return false
	}

	// Take the lock
	if !w.lock.TryLock() {
		return false
	}
	defer w.lock.Unlock()

	c := w.configContext

	c.userRoot = ref
	c.userRootIsDirty = true
	w.dirty.Store(true)

	return true
}

func (s *ConfigContextService) GetCurrent(handle ConfigContextHandle) *ConfigVersionRef {
	w := s.getWrapperInternal(handle)
	if w == nil {
		s.logger.Printf("GetCurrent: Invalid handle\n")
		return nil
	}

	return w.configContext.current
}

func (s *ConfigContextService) SetCurrent(handle ConfigContextHandle, parentRef *ConfigVersionRef, ref *ConfigVersionRef) bool {
	w := s.getWrapperInternal(handle)
	if w == nil {
		s.logger.Printf("SetCurrent: Invalid handle\n")
		return false
	}

	// Take the lock
	if !w.lock.TryLock() {
		return false
	}
	defer w.lock.Unlock()

	c := w.configContext

	c.current = ref
	c.currentIsDirty = true
	if parentRef != nil {
		c.parent = parentRef
	}
	w.dirty.Store(true)

	return true
}

// func (s *ConfigContextService) NextVersion(ctx context.Context, tx *gorm.DB, configCtxHandle ConfigContextHandle, accountId util.AccountId, userId *util.UserId, parentRef *ConfigVersionRef, dataObject *ConfigDataObject) (*ConfigVersionRef, error) {
// 	w := s.getWrapperInternal(configCtxHandle)
// 	if w == nil {
// 		s.logger.Printf("NextVersion: Invalid handle\n")
// 		return nil, nil
// 	}

// 	// Create a new version
// 	newVersion, err := s.versionService.allocateVersionInternal(ctx, tx, w, accountId, userId, parentRef, dataObject, nil)
// 	if err != nil {
// 		s.logger.Printf("NextVersion: Error creating new version for account %s and user %s: %v\n", accountId, userId, err)
// 		return nil, err
// 	}

// 	// Set the current version
// 	if !s.SetCurrent(configCtxHandle, parentRef, newVersion) {
// 		s.logger.Printf("NextVersion: Error setting current version for account %s and user %s\n", accountId, userId)
// 		return nil, nil
// 	}

// 	return newVersion, nil
// }

func (s *ConfigContextService) save(ctx context.Context, tx *gorm.DB, w *configContextWrapper, ignoreDirty bool) (bool, error) {
	// Take the lock
	if !w.lock.TryLock() {
		s.logger.Printf("Save: could not lock\n")
		return false, nil
	}
	defer w.lock.Unlock()

	if !ignoreDirty && !w.dirty.Load() {
		s.logger.Printf("Save: not dirty\n")
		return true, nil
	}

	if w.configContext == nil {
		s.logger.Printf("Save: Invalid handle, missing config context\n")
		return false, nil
	}

	configCtx := w.configContext
	accountId := configCtx.accountId
	userId := configCtx.userId

	// NOTE: We should only be upserting the refs and failing
	// if the version does not exist in the database.
	// *Only* committing an object should create a new version.
	// This also means we only need skipDB as an optimization, not
	// to prevent duplicate version records.

	// Save to database
	err := util.WithTransaction(s.db, tx, func(tx *gorm.DB) error {
		saveRef := func(scope util.ScopeKind, kind ConfigReferenceKind, ref *ConfigVersionRef, skipDb bool) error {
			// // Insert version
			// if err := s.versionService.createConfigVersionRecord(tx, accountId, userId, nil, *ref); err != nil {
			// 	s.logger.Printf("saveVersionRef: Error saving config version for account %s and user %s: %v\n", accountId, userId, err)
			// 	return err
			// }

			// Upsert ref
			// TODO: Fail if the version does not exist
			return s.refService.SetConfigReference(ctx, tx, scope, accountId, userId, kind, ref)
		}

		if configCtx.accountRootIsDirty && !configCtx.accountRootSkipDb {
			if err := saveRef(util.ScopeKindAccount, ConfigReferenceKindRoot, configCtx.accountRoot, false); err != nil {
				return err
			}
		}

		if configCtx.userRootIsDirty && !configCtx.userRootSkipDb {
			if err := saveRef(util.ScopeKindUser, ConfigReferenceKindRoot, configCtx.userRoot, false); err != nil {
				return err
			}
		}

		if configCtx.currentIsDirty && !configCtx.currentSkipDb {
			if err := saveRef(util.ScopeKindAccount, ConfigReferenceKindHead, configCtx.current, false); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		s.logger.Printf("saveVersionRef: Error saving config version for account %s and user %s to database: %v\n", accountId, userId, err)
		return false, err
	}

	// Clear dirty flag
	w.dirty.Store(false)

	return true, nil
}

func (s *ConfigContextService) Save(ctx context.Context, tx *gorm.DB, handle ConfigContextHandle) (bool, error) {
	w := s.getWrapperInternal(handle)
	if w == nil {
		s.logger.Printf("Save: Invalid handle\n")
		return false, nil
	}

	// Check if dirty
	if !w.dirty.Load() {
		return true, nil
	}

	return s.save(ctx, tx, w, false)
}
