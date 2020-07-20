# Just three quick steps to start the server

### step 1. Clone

git clone https://github.com/pion/ion
cd ion

### step 2.Creates a new network. 
docker network create ionnet

### step 3.Start all modules
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
