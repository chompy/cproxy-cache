CProxy-Cache By Nathan Ogden
============================

About
-----

CProxy-Cache is an extension to [CProxy](https://gitlab.com/chompy/cproxy). It adds a hybrid memory and filesytem
HTTP cache to CProxy with ESI support.


Building
--------

```
go get github.com/pquerna/cachecontrol/cacheobject
go build -buildmode=plugin
```

This will create a '.so' or '.dll' file depending on your OS. Add this file to the 'ext' directory with the
CProxy executable and add the filename to the extension list in 'cproxy.json.'