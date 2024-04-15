import http from 'k6/http';

import { ApiClient, USER_EMAIL, USER_PASSWORD, USER_ACCOUNT_ID } from './lib/api.js';

export const options = {
    vus: 1,
    iterations: 1,
};

const configKey = 'testConfigKey';

class TestFailed extends Error {
    constructor(message) {
        super(`Test failed: ${message}`);
        this.name = 'TestFailed';
    }
}

const compareObjects = (a, b) => {
    if (Object.keys(a).length !== Object.keys(b).length) {
        return false;
    }

    for (const key of Object.keys(a)) {
        const [ak, bk] = [a[key], b[key]];
        if (typeof ak === 'object' && typeof bk === 'object') {
            if (!compareObjects(ak, bk)) {
                return false;
            }
        } else if (ak !== bk) {
            return false;
        }
        return true;
    }

    return true;
};

export default async function () {

    const baseUrl = 'https://api.beta.config_api.io';

    const api = new ApiClient(baseUrl, http);

    const token = await api.login(USER_EMAIL, USER_PASSWORD, USER_ACCOUNT_ID);

    console.info(`Token: ${token}`);

    const putData = async (data) => {
        const res = await api.putKeyedConfigSettings(USER_ACCOUNT_ID, undefined, configKey, data);
        const versionHash = res.headers['X-Config-Version-Hash'];
        console.info(`Version hash: ${versionHash}`);
        const versionId = res.headers['X-Config-Version-Id'];
        console.info(`Version id: ${versionId}`);
        console.info(`Response: ${JSON.stringify(res)}`);
        return versionHash;
    };

    // await api.putKeyedConfigSettings(USER_ACCOUNT_ID, undefined, configKey, { value: 'testConfigValue' });

    const hash1 = await putData({ value: 'testConfigValue1' });

    const hash2 = await putData({ value: 'testConfigValue2' });

    const expectedDiff2 = Object.freeze([{
        "path": "/value",
        "value": "testConfigValue2",
        "op": "replace"
    }]);

    const diffsResp = await api.getConfigResourceDiffsBetweenHashes(USER_ACCOUNT_ID, undefined, `configs/${configKey}`, hash1, hash2);
    if (diffsResp.status !== 200) {
        throw new TestFailed(`Failed to get diffs: ${diffsResp.status} ${diffsResp.statusText}`);
    }

    const diffs = diffsResp.json();
    console.info(`Diffs: ${JSON.stringify(diffs, null, 2)}`);

    const v2 = Object.values(diffs.versions).filter(diff => diff.to_version && diff.to_version.config_version_hash === hash2)[0];
    if (!v2) {
        throw new TestFailed('Failed to find diff for hash2');
    }
    const v2diff = v2.record_diff;
    if (!v2diff) {
        throw new TestFailed('Failed to find record_diff for hash2');
    }

    console.info(`Diff for hash2: ${JSON.stringify(v2diff, null, 2)}`);
    console.info(`Expected diff for hash2: ${JSON.stringify(expectedDiff2, null, 2)}`);

    if (compareObjects(v2diff, expectedDiff2)) {
        console.info('Diffs match');
    } else {
        throw new TestFailed('Diffs do not match comparing actual and expected');
    }

}
