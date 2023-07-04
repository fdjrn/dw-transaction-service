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
    - mdw.transaction.topup.result          ✅
    - mdw.transaction.deduct.result
    - mdw.transaction.transfer.result

### Published/Produced Topic
    - mdw.transaction.topup.request         ✅
    - mdw.transaction.deduct.request
    - mdw.transaction.transfer.request
    

### RESTFul API Endpoint
    - POST /api/v1/transaction/topup        ✅
    - POST /api/v1/transaction/deduct
    - POST /api/v1/transaction/distribute
    - POST /api/v1/transaction/inquiry      ✅