# Serverless
This example shows how to deploy a basic API to AWS using Serverless and the [eawsy Lambda Go shim](https://github.com/eawsy/aws-lambda-go-shim). A basic function handler is defined in `handler.go` and an API with a single HTTP event is defined in `serverless.yml`. There is nothing specific to Chop with this approach, we simply build a zip package and specify it as the artifact in the YAML definition.

## Getting started
The following steps can be used to deploy the API to AWS. **Please be aware that this may incur AWS costs**.

* Install Serverless using `npm install -g serverless`.
* Create a set of AWS access keys and [configure them for use with Serverless](https://serverless.com/framework/docs/providers/aws/guide/credentials/).
* Navigate to the example directory and run `wget -O Makefile https://git.io/vytH8` to get the latest eawsy Makefile.
* Run `make` to build and package the handler.
* Run `serverless deploy` to deploy the API artifact and CloudFormation.
* Check that everything works as expected then query the API endpoint returned by Serverless.