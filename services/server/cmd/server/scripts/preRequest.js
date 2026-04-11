const now = Date.now();
const expiry = pm.variables.get("token_expiry");
const existingToken = pm.variables.get("bearerToken");

const applyToken = (token) => {
    pm.collectionVariables.set("bearerToken", token);
    pm.request.headers.upsert({ key: "Authorization", value: "Bearer " + token });
};

// Reuse existing token if still valid
if (existingToken && expiry && now < parseInt(expiry)) {
    applyToken(existingToken);
    return;
}

const baseUrl = pm.variables.get("baseUrl") || pm.variables.get("base_url");
const clientId = pm.variables.get("client_id");
const clientSecret = pm.variables.get("client_secret");

if (!clientId || !clientSecret) {
    console.log("Skipping token refresh: client_id or client_secret not set");
    return;
}

pm.sendRequest({
    url: baseUrl + "/token",
    method: "POST",
    header: { "Content-Type": "application/json" },
    body: {
        mode: "raw",
        raw: JSON.stringify({
            grant_type: "client_credentials",
            client_id: clientId,
            client_secret: clientSecret
        })
    }
}, (err, res) => {
    if (err) {
        console.error("Token refresh failed:", err);
        return;
    }
    const body = res.json();
    applyToken(body.access_token);
    // Refresh 60 seconds before expiry
    pm.collectionVariables.set("token_expiry", String(now + (body.expires_in - 60) * 1000));
});
