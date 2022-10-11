# Test invalid HTTPS certificates

See [What you need to know about HTTPS Common Name deprecation in OpenShift 4.10](https://cloud.redhat.com/blog/details-on-https-common-name-deprecation-in-openshift-4.10).

```shell
go build .
export OS_CLOUD=<my cloud>
./openstack-invalid-https-cert-scanner
```
