# DNSKEY Input Plugin

The DNSKEY plugin gathers dnskey data - like [Dig](https://en.wikipedia.org/wiki/Dig_\(command\))

### Configuration:

```
# Sample Config:
[[inputs.dnskey]]
  ## Domains to query.
  # domains = ["ietf.org", "icann.org"]

  ## Resolvers (to specify port write ipv4:53 or [ipv6]:53)
	# resolvers = ["8.8.8.8", "8.8.4.4"]

  ## Query timeout in seconds.
  # timeout = 2

```

### Tags:

- server
- domain
- algorithm
- key type
- keytag

### Example output:

```
telegraf --input-filter dns_query --test
> dnskey,domain=iis.se,server=8.8.8.8:53,keytag=18937,algorithm=RSASHA1,key_type=KSK,host=localhost query_time_ms="18.086996ms" 1504708135000000000
> dnskey,domain=iis.se,server=8.8.8.8:53,keytag=46005,algorithm=RSASHA1,key_type=ZSK,host=localhost query_time_ms="18.086996ms" 1504708135000000000
> dnskey,domain=ietf.org,server=8.8.8.8:53,keytag=45586,algorithm=RSASHA1,key_type=KSK,host=localhost query_time_ms="17.229442ms" 1504708135000000000
> dnskey,domain=ietf.org,server=8.8.8.8:53,keytag=40452,algorithm=RSASHA1,key_type=ZSK,host=localhost query_time_ms="17.229442ms" 1504708135000000000
```
