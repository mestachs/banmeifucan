
think dockerize but for a proxy

go build -o reverse_proxy .


TODO
  - test it on a real server ;)
  
  - add a token to unban
  - make threshold and banDuration adjustable
  - keep track of other errors status (409, 50x)- 
  - detect if ip is from a hosting
    - AWS : https://ip-ranges.amazonaws.com/ip-ranges.json
    - GCP : https://www.gstatic.com/ipranges/cloud.json
    - list of other hosting services : https://github.com/femueller/cloud-ip-ranges/tree/master