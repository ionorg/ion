# Authentication

Ion supports JWT based authorization. Token issuance and verification is not handled by Ion.

To enable authorization, set the `authorization` value in the `biz` config.

```toml
[signal]
authorization = true
```

JWT tokens should be passed using an `access_token` query parameter in the websocket connection url to `biz`. Each token should contain a claim with room permissions:

```json
{
  "id": "...", // room id
  "videopublish": true, // can a peer publish video
  "audiopublish": true // can a peer publish audio
}
```
