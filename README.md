# s3browser_anon
A simple Golang web service to browse and search S3 buckets

To use as-is:
* Replace S3ENDPOINT environment variable with appropriate value.
* Create a secret with S3 access keys
```kubectl create secret generic my-s3-keys --from-literal=access-key='XXXXXXX' --from-literal=secret-key='YYYYYYY'
* Deploy using deployment.yaml

The application is currently exposing as a ClusterIP service, so that will need to be modified to expose the application outside of Kubernetes.

To update source code:
* Modify server.go and use the included Dockerfile to rebuild
