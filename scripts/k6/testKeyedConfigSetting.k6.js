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

export default async function() {

    const baseUrl = 'https://api.beta.config_api.io';

    const api = new ApiClient(baseUrl, http);

    const token = await api.login(USER_EMAIL, USER_PASSWORD, USER_ACCOUNT_ID);

    console.info(`Token: ${token}`);

    const putData = async (data) => {
        const res = await api.putKeyedConfigSettings(USER_ACCOUNT_ID, undefined, configKey, data);
        const checkHash = res.headers['X-Config-Version-Hash'];
        console.info(`Version hash: ${checkHash}`);
        const versionId = res.headers['X-Config-Version-Id'];
        console.info(`Version id: ${versionId}`);
        console.info(`Response: ${JSON.stringify(res)}`);
        return checkHash;
    };

    const value1 = Object.freeze({ value: 'testConfigValue1' });

    const hash1 = await putData(value1);

    const [check1, checkHash1] = await api.getKeyedConfigSettings(USER_ACCOUNT_ID, undefined, configKey);

    console.info(`Value1:     ${JSON.stringify(value1)}`);
    console.info(`Check1:     ${JSON.stringify(check1)}`);
    console.info(`Hash1:      ${hash1}`);
    console.info(`CheckHash1: ${checkHash1}`);

    if (hash1 === checkHash1) {
        console.info('Hashes match');
    } else {
        throw new TestFailed('Hashes do not match comparing hash1 and checkHash1');
    }

    if (JSON.stringify(check1) === JSON.stringify(value1)) {
        console.info('Values match');
    } else {
        throw new TestFailed('Values do not match comparing check1 and value1');
    }

    const value2 = Object.freeze({ value: 'testConfigValue2' });

    const hash2 = await putData(value2);

    const [check2, checkHash2] = await api.getKeyedConfigSettings(USER_ACCOUNT_ID, undefined, configKey);

    console.info(`Value2:     ${JSON.stringify(value2)}`);
    console.info(`Check2:     ${JSON.stringify(check2)}`);
    console.info(`Hash2:      ${hash2}`);
    console.info(`CheckHash2: ${checkHash2}`);

    if (hash2 === checkHash2) {
        console.info('Hashes match');
    } else {
        throw new TestFailed('Hashes do not match comparing hash2 and checkHash2');
    }

    if (JSON.stringify(check2) === JSON.stringify(value2)) {
        console.info('Values match');
    } else {
        throw new TestFailed('Values do not match comparing check2 and value2');
    }

}
