

Think [dockerize](https://github.com/jwilder/dockerize) but for a proxy to ban undesired traffic (too much 404 for the moment)

# dev

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
  - [x] make threshold and banDuration adjustable
  - [x] keep track of other errors status (409, 50x)- 
  - [ ] allow whitelist
  - [ ] detect if ip is from a hosting
    - AWS : https://ip-ranges.amazonaws.com/ip-ranges.json
    - GCP : https://www.gstatic.com/ipranges/cloud.json
    - list of other hosting services : https://github.com/femueller/cloud-ip-ranges/tree/master


# goreleaser

```
git tag -a v0.1.1 -m "Second release"
git push origin v0.1.1
GITHUB_TOKEN=.... goreleaser
```