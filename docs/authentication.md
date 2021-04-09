# Authentication

Ion supports JWT based authorization. Token issuance is not handled by ION, but the token signature is validated using the key set in the config.  Both connection and message level authentication is supported (for rooms).

To enable authentication, set enabled to true in the config for the type you want

```toml
[signal.auth_connection]
enabled = true 
key_type = "HMAC"  
key = "$HMAC_SECRET"

[signal.auth_room]
enabled = true
key_type = "HMAC"
key = "$HMAC_SECRET"
```

*Currently only HMAC is supported, but new key types can be added in the future*


## Connection Auth
JWT tokens should be passed using an `access_token` query parameter in the websocket connection url to `biz`.  This does not currently validate any claims, just that the token has a valid signature. 



## Room Auth
JWT Tokens should be passed as a `"token"` parameter on the JoinMsg and Publish requests via websocket.  
```json
{
  "sid": "...", // session id
  "videopublish": true, // can a peer publish video
  "audiopublish": true // can a peer publish audio
}
```
