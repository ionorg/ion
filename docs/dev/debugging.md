## How to develop and debug server module
### Step 1:
Stop the module, like:
```
./scripts/bizStop.sh
```
and check biz is stopped
```
ps -ef|grep biz
```
You can modify the code then
### Step 2:
Rebuild the module and run:
```
./scripts/bizStart.sh
```

