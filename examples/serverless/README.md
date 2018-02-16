# Serverless
This example shows how to deploy a function using Serverless, API Gateway and the Go Lambda runtime. A basic handler is defined in `main.go` and an API with a proxy integration event is defined in `serverless.yml`.

## Getting started
The following steps can be used to deploy the function to AWS. **Please be aware that this may incur AWS costs**.

* Install Serverless using `npm install -g serverless`.
* Create a set of AWS access keys and [configure them for use with Serverless](https://serverless.com/framework/docs/providers/aws/guide/credentials/).
* Run `make deploy` to build and deploy the handler
* Check that everything works as expected then query the API endpoint returned by Serverless
* Run `make remove` to delete the CloudFormation stack and clean the working directory