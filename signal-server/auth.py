import jwt
import time

claims = { "sub":"room1","exp": int(time.time()) + 3600*24*365}
token = jwt.encode(claims, "secret", algorithm="HS256").decode()
print(token)

