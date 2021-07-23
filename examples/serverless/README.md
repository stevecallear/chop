# Serverless
This example demonstrates how to deploy a single function using Serverless that handles API Gateway proxy integration, API Gateway HTTP V2 and ALB target group events. The basic handler is defined in `main.go` and the event handlers are specified in `serverless.yml`.

## Deploy
The following steps can be used to deploy the function to AWS. **Please be aware that this may incur AWS costs**.

* Install Serverless using `npm install -g serverless`.
* Create a set of AWS access keys and [configure them for use with Serverless](https://serverless.com/framework/docs/providers/aws/guide/credentials/).
* Run `make deploy` to build and deploy the handler

## Invoke
The Lambda function can be invoked via either the ALB DNS record or the API Gateway endpoints.

* Copy the API Gateway endpoint from the serveress output
* Invoke the function using API Gateway proxy integration events, e.g. `curl https://{value}.execute-api.eu-west-1.amazonaws.com/dev/resource?a=1`
* Invoke the function using API Gateway HTTP V2 events, e.g. `curl https://{value}.execute-api.eu-west-1.amazonaws.com/resource?a=1`
* Get the ALB DNS, e.g. `aws cloudformation describe-stacks --region eu-west-1 --stack-name chop-example-dev | grep elb.amazonaws.com`
* Invoke the function using the ALB Target Group, e.g. `curl chop-example-dev-{value}.eu-west-1.elb.amazonaws.com/resource?a=1`

## Remove
* Run `make remove` to delete the CloudFormation stack and clean the working directory