
const ROOT_ACCOUNT_ID = '00000000-0000-0000-0000-000000000000';

export const login = async (http, baseUrl, accountId = ROOT_ACCOUNT_ID, email = '', password = '') => {
    email = email || 'root@test.tld';
    password = password || 'password';

    const tokenBody = {
        email: email,
        password: password,
    };

    const accountUrl = accountId ? `${baseUrl}/accounts/${accountId}` : `${baseUrl}/platform/`;

    const tokenResp = http.post(`${accountUrl}/auth/tokens/`, JSON.stringify(tokenBody), {
        headers: {
            'Content-Type': 'application/json',
        },
    });

    console.info('Response: ' + tokenResp.body);

    const token = tokenResp.json('token');
    console.info(`Token: ${token}`);

    return token;
}