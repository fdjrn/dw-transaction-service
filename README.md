# Transaction Service
---
### Deskripsi
Service ini berfungsi untuk meng-handle transaksi saldo pengguna, seperti:

    - Topup Saldo
    - Deduct Saldo
    - Transfer / Distribusi Saldo
    
### Service Type
    - Message Producer
    - Message Consumer
    - RestAPI endpoint

### Subscribed/Consumed Topic
    - mdw.transaction.topup.result              ✅
    - mdw.transaction.deduct.result             ✅
    - mdw.transaction.distribute.result         ✅
    - mdw.transaction.distribute.result.members ✅

### Published/Produced Topic
    - mdw.transaction.topup.request             ✅
    - mdw.transaction.deduct.request            ✅
    - mdw.transaction.distribute.request        ✅
    

### RESTFul API Endpoint
    - POST /api/v1/transaction/topup            ✅
    - POST /api/v1/transaction/deduct           ✅
    - POST /api/v1/transaction/distribute       ✅
    - POST /api/v1/transaction/inquiry          ✅
    - POST /api/v1/transaction/summary          ✅
    - POST /api/v1/transaction/summary/topup    ✅
    - POST /api/v1/transaction/summary/deduct   ✅


### Build Docker Image
    docker build -t dw-transaction:1.0.0 -f Dockerfile .

### Available Environment Value:
    - DATABASE_MONGODB_URI : conncetion uri to mongodb cluster
        
        example: mongodb+srv://<user>:<password>@<cluster-host>/?retryWrites=true&w=majority

    - DATABASE_MONGODB_DB_NAME : Database Name used for parameter service

        example: dw-mdw-transaction

    - KAFKA_BROKERS : kafka cluster address

        example: touching-ghoul-8389-us1-kafka.upstash.io:9092

    - KAFKA_SASL_USER : kafka cluster username

    - KAFKA_SASL_PASSWORD : kafka cluster password

### Docker Run Command
    docker run -d -p 8100:8100 --name dw-transaction-service --env "DATABASE_MONGODB_DB_NAME=dev-mdw-transaction" --restart unless-stopped dw-transaction:1.0.0