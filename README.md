

Think [dockerize](https://github.com/jwilder/dockerize) but for a proxy to ban undesired traffic (too much 404 for the moment)


# Motivation

We had that small server getting scanned over and over, leading to downtime, extra usage,... 
For a nodejs app we didn't code, and really didn't know all the inner details.

We tried various thing but not something really fitting this basic use case 
 - a bot come in scan tons of url triggering plenty of 404 
 - sometimes triggering some endpoint that were calling a db or a redis, taking a bit more resource/time to answer 404

It's hard to say if these bot are all malcious or just "crawling" the internet but some clearly were bad one.


# Dev

## Local development

```
go build -o reverse_proxy .
```

or 

```
./script/test.sh
```

testing locally


```

curl -vvv -H "X-Forwarded-For: 192.168.1.100" http://127.0.0.1:8000/fonts/456454 

ali http://127.0.0.1:8000/fonts/456454
```


TODO
  - [x]test it on a real server ;)
  - [ ] add a token to unban
  - [x] add a small dashboard
  - [x] add tracking of percentiles by path (handle rest api in a certain form)
  - [x] make threshold and banDuration adjustable
  - [x] keep track of other errors status (409, 50x)- 
  - [ ] allow whitelist
  - [ ] allow predefined rules (ban .env, php, java,...)
  - [ ] detect if ip is from a hosting
    - AWS : https://ip-ranges.amazonaws.com/ip-ranges.json
    - GCP : https://www.gstatic.com/ipranges/cloud.json
    - list of other hosting services : https://github.com/femueller/cloud-ip-ranges/tree/master


## Release

to release, I setuped [goreleaser](https://goreleaser.com/quick-start/) so artifacts end up in released package of the repo

```
git describe --tags --abbrev=0
export NEW_TAG=v0.1.5
rm -rf ./dist
git tag -a $NEW_TAG -m "Second release"
git push origin $NEW_TAG
GITHUB_TOKEN=.... goreleaser
```
