import http from 'k6/http';

import { ApiClient, USER_EMAIL, USER_PASSWORD, USER_ACCOUNT_ID } from './lib/api.js';

export const options = {
    vus: 1,
    iterations: 1,
};

// const configKey = 'testConfigKey';

class TestFailed extends Error {
    constructor(message) {
        super(`Test failed: ${message}`);
        this.name = 'TestFailed';
    }
}

export default async function() {

    const baseUrl = 'https://api.beta.config_api.io';

    const api = new ApiClient(baseUrl, http);

    const token = await api.login(USER_EMAIL, USER_PASSWORD, USER_ACCOUNT_ID);

    console.info(`Token: ${token}`);

    const putData = async (configKey, data) => {
        const res = await api.putKeyedConfigSettings(USER_ACCOUNT_ID, undefined, configKey, data);
        const checkHash = res.headers['X-Config-Version-Hash'];
        console.info(`Version hash: ${checkHash}`);
        const versionId = res.headers['X-Config-Version-Id'];
        console.info(`Version id: ${versionId}`);
        console.info(`Response: ${JSON.stringify(res)}`);
        return checkHash;
    };

    const value1 = Object.freeze({ key1: 'testConfigValue1' });

    const hash1 = await putData('ConfigKey1', value1);

    const value2 = Object.freeze({ key2: 'testConfigValue2' });

    const hash2 = await putData('ConfigKey2', value2);

    const value1v2 = Object.freeze({ key1: 'testConfigValue1v2' });

    const hash1v2 = await putData('ConfigKey1', value1v2);

    const hash3 = await putData('ConfigKey3', { key3: 'testConfigValue3' });

    const hash4 = await putData('ConfigKey4', { key4: 'testConfigValue4' });

    const recordListResp = await api.getConfigRecordList(USER_ACCOUNT_ID, undefined);

    const recordList = recordListResp.json();

    console.info(`RecordList: ${JSON.stringify(recordList, null, 2)}`);

    const getKeyedRecordHash = async (configKey) => {
        const entry = recordList.find(entry => entry.record_collection_key === configKey);
        console.info(`Entry: ${JSON.stringify(entry, null, 2)}`);
        if (!entry) {
            throw new TestFailed(`No entry found for ${configKey}`);
        }
        return entry.node_metadata.version_ref.config_version_hash;
    }

    const actualHash1 = await getKeyedRecordHash('ConfigKey1');
    const actualHash2 = await getKeyedRecordHash('ConfigKey2');
    const actualHash3 = await getKeyedRecordHash('ConfigKey3');

    console.info(`Hash1: ${hash1}`);
    console.info(`Hash1v2: ${hash1v2}`);
    console.info(`ActualHash1: ${actualHash1}`);
    console.info(`Hash2: ${hash2}`);
    console.info(`ActualHash2: ${actualHash2}`);
    console.info(`Hash3: ${hash3}`);
    console.info(`ActualHash3: ${actualHash3}`);

    if (hash1v2 === actualHash1) {
        console.info('Hashes match');
    } else {
        throw new TestFailed('Hashes do not match comparing hash1v2 and actualHash1');
    }

    if (hash2 === actualHash2) {
        console.info('Hashes match');
    } else {
        throw new TestFailed('Hashes do not match comparing hash2 and actualHash2');
    }

    if (hash3 === actualHash3) {
        console.info('Hashes match');
    } else {
        throw new TestFailed('Hashes do not match comparing hash3 and actualHash3');
    }
}
