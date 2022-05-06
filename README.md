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

### Source Compile

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

### Docker
* docker >= 20.10
* docker-compose >= 2.2.2

```bash
sudo curl -L "https://github.com/docker/compose/releases/download/v2.2.2/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
sudo ln -s /usr/local/bin/docker-compose /usr/bin/docker-compose
docker-compose up -d
```

_if you already have a mysql installed, just run_
```bash
docker run -dp 8125-8126:8125-8126 -v $PWD/config/config.yaml:/app/config/config.yaml --name sub-account-server slagga/sub-account
```

### Others
More APIs see [API.md](https://github.com/dotbitHQ/sub-account-svr/blob/main/API.md)
