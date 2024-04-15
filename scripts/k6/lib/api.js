
export const DEFAULT_EMAIL = 'root@test.tld';
export const DEFAULT_PASSWORD = 'password';
export const DEFAULT_ACCOUNT_ID = '00000000-0000-0000-0000-000000000000';

export const USER_EMAIL = 'stripetest4@test.tld';
export const USER_PASSWORD = 'testpass';
export const USER_ACCOUNT_ID = '94925f31-1e72-4c1e-b096-8ed4febba824';

export class ApiClient {
    baseUrl;
    http;
    
    _token;

    constructor(baseUrl, http) {
        this.baseUrl = baseUrl;
        this.http = http;
    }

    set token(token) {
        this._token = token;
    }

    get token() {
        return this._token;
    }

    makeHeaders(extraHeaders = {}) {
        const headers = {
            'Content-Type': 'application/json',
        }
        for (const [key, value] of Object.entries(extraHeaders)) {
            headers[key] = value;
        }
        if (this._token) {
            headers['Authorization'] = `Bearer ${this._token}`;
        }
        return headers;
    }

    async makeRequest(method, path, body, extraHeaders = {}) {
        const headers = this.makeHeaders(extraHeaders);
        console.info(`${method} ${this.baseUrl}${path} with headers: ${JSON.stringify(headers)}`);
        const resp = this.http.request(method, `${this.baseUrl}${path}`, body, { headers });
        console.info(`Status: ${resp.status} ${resp.statusText}`);
        // try {
        //     const body = resp.json();
        //     console.info(`Response: ${JSON.stringify(body)}`);
        //     return body;
        // } catch (e) {
        //     console.info('Error parsing response');
        //     return null;
        // }
        return resp;
    }

    async getJson(path, extraHeaders = {}) {
        return this.makeRequest('GET', path, null, extraHeaders);
    }

    async postJson(path, body, extraHeaders = {}) {
        return this.makeRequest('POST', path, JSON.stringify(body), extraHeaders);
    }

    async putJson(path, body, extraHeaders = {}) {
        return this.makeRequest('PUT', path, JSON.stringify(body), extraHeaders);
    }

    async deleteJson(path, extraHeaders = {}) {
        return this.makeRequest('DELETE', path, null, extraHeaders);
    }

    async login(email = DEFAULT_EMAIL, password = DEFAULT_PASSWORD, accountId = DEFAULT_ACCOUNT_ID) {
        const tokenBody = {
            email: email,
            password: password,
        };

        const apiUrl = `/accounts/${accountId}/auth/tokens/`;

        const tokenResp = await this.postJson(apiUrl, tokenBody);

        console.info('Status: ' + tokenResp.status);
        console.info('Status text: ' + tokenResp.statusText);

        console.info('Response: ' + tokenResp.body);

        const token = tokenResp.json('token');
        console.info(`Token: ${token}`);

        this._token = token;

        return token;
    }

    async createProduct(accountId, product) {
        return this.postJson(`/accounts/${accountId}/products/`, product);
    }

    async createCheckoutToken(accountId, checkoutToken) {
        console.info(`Creating checkout token for account ${accountId}`);
        return this.postJson(`/accounts/${accountId.toString()}/checkout_tokens/`, checkoutToken);
    }

    async createPurchaseRequest(accountId, purchaseRequest) {
        return this.postJson(`/accounts/${accountId.toString()}/purchase_requests/`, purchaseRequest);
    }

    async createCheckoutTransaction(checkoutTransaction) {
        return this.postJson('/checkout_transactions/', checkoutTransaction);
    }

    // TODO: Make configKey and itemKey optional
    async getConfigRecordList(accountId, userId) {
        const userPart = userId ? `/users/${userId.toString()}` : '';
        const apiUrl = `/accounts/${accountId.toString()}${userPart}/configs`;

        console.info(`Getting config record list for account ${accountId}, user ${userId}`);

        return this.getJson(apiUrl);
    }

    async getKeyedConfigSettings(accountId, userId, configKey) {
        const userPart = userId ? `/users/${userId.toString()}` : '';
        const apiUrl = `/accounts/${accountId.toString()}${userPart}/configs/${configKey}`;

        console.info(`Getting config settings for account ${accountId}, user ${userId}, key ${configKey}`);

        const resp = await this.getJson(apiUrl);
        const versionHash = resp.headers['X-Config-Version-Hash'];

        const respBody = resp.json();
        console.info(`Response body: ${JSON.stringify(respBody)}`);

        return [respBody, versionHash];
    }

    async putKeyedConfigSettings(accountId, userId, configKey, configValues) {
        const userPart = userId ? `/users/${userId.toString()}` : '';
        const apiUrl = `/accounts/${accountId.toString()}${userPart}/configs/${configKey}`;

        console.info(`Setting config settings for account ${accountId}, user ${userId}, key ${configKey} with values: `, configValues);

        return this.putJson(apiUrl, configValues);
    }

    async getConfigVersionsBetweenHashes(accountId, userId, fromHash, toHash) {
        const userPart = userId ? `/users/${userId.toString()}` : '';
        // const apiUrl = `/accounts/${accountId.toString()}${userPart}/configs/${configKey}/diffs?from=${fromHash}&to=${toHash}`;
        const apiUrl = `/accounts/${accountId.toString()}${userPart}/config/diff/versions/${fromHash}/${toHash}`;

        console.info(`Getting config diffs for account ${accountId}, user ${userId} between hashes ${fromHash} and ${toHash}`);

        return this.getJson(apiUrl);
    }

    async getConfigResourceDiffsBetweenHashes(accountId, userId, resourcePath, fromHash, toHash) {
        const userPart = userId ? `/users/${userId.toString()}` : '';
        // const apiUrl = `/accounts/${accountId.toString()}${userPart}/configs/${configKey}/diffs?from=${fromHash}&to=${toHash}`;
        const apiUrl = `/accounts/${accountId.toString()}${userPart}/config/diff/${resourcePath}/${fromHash}/${toHash}`;

        console.info(`Getting config diffs for account ${accountId}, user ${userId}, resource path ${resourcePath} between hashes ${fromHash} and ${toHash}`);

        return this.getJson(apiUrl);
    }

}
