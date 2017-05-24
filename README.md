## Simple TCP Proxy that fragmentates TCP data into bytes

A small tool to split TCP data into few bytes and send them with delay.
This could be useful when debuging proto parsing codes.

### Usage
```
Usage of ./byteproxy:
  -d int
        delay in ms
  -l string
        listen port
  -r string
        throttle direction: cs | sc | both | none
        * cs: throttle client to server data path
        * sc: throttle server to client data path
        * both: throttle both direction (default)
        * none: do not throttle
        (default "both")
  -s int
        number of bytes per TCP packet after throttled (default 1)
  -u string
        upstream port
```


## Example

### Split and delay

```
./byteproxy -l :8080 -u :8081 -s 1 -d 1000 -r cs
```

Split tcp data into one byte per IP packet, and add delay for 1 second.
the `-r cs` option means only apply to the direction from client to upstream
server, so the data replied from upstream is returned untouched.

To split and delay both direction, use `-r both`.
