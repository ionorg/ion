# Just two quick steps to start the server

### step 1.Creates a new network. 
docker network create ionnet

### step 2.Start all modules
./scripts/allStart.sh



# Other commands

### 1. Stop all modules
./scripts/allStop.sh
### 2. Restart all modules
./scripts/allRestart.sh
### 3. Invidial module scripts
./scripts/bizStart.sh

./scripts/bizStop.sh

./scripts/islbStart.sh

./scripts/islbStop.sh

./scripts/sfuStart.sh

./scripts/sfuStop.sh
