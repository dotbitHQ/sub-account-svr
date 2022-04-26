# sub-account-svr
Backend of .bit sub account service, including registration and management. 

# Prerequisites

* Ubuntu 18.04 or newer (2C4G)
* MYSQL >= 8.0
* Redis >= 5.0 (for cache)
* GO version >= 1.16.15
* Mongo >= 4.2 (2 Cores, 4 GB, 200 G Disk Space)
* [CKB Node](https://github.com/nervosnetwork/ckb)
* [das-database](https://github.com/dotbitHQ/das-database)

## Install & Run

```bash
# get the code
git clone https://github.com/dotbitHQ/sub-account-svr

# rename config/config.example.yaml to config/config.yaml, then edit config/config.yaml before init mysql database
mysql -uroot -p
> source sub-account-svr/tables/sub_account_db.sql;
> quit;

# compile and run
cd sub-account-svr
make sub
./sub_account --config=config/config.yaml
```

## Docker Install & Run
```bash
# if you already have a mysql and mongo database installed, just run
docker run -dp 8125-8126:8125-8126 -v $PWD/config/config.yaml:/app/config/config.yaml --name sub-account-server slagga/sub-account

# if not, you need docker-compose to automate the installation
docker-compose up -d
```

### Others
More APIs see [API.md](https://github.com/dotbitHQ/sub-account-svr/blob/main/API.md)
