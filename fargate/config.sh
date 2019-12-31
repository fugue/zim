#!/bin/bash

# This is a bit lame but works fine. It extracts output values from the Zim
# CloudFormation stack and prints a JSON object with the info to stdout.
# The output is intended to be saved to ~/.zim/config.json.

CLUSTER=$(aws cloudformation describe-stacks \
    --stack-name zim \
    --output text \
    --query 'Stacks[0].Outputs[?OutputKey==`ClusterName`].OutputValue')

TASKDEF_GO=$(aws cloudformation describe-stacks \
    --stack-name zim \
    --output text \
    --query 'Stacks[0].Outputs[?OutputKey==`TaskDefinitionGo`].OutputValue')

TASKDEF_PYTHON=$(aws cloudformation describe-stacks \
    --stack-name zim \
    --output text \
    --query 'Stacks[0].Outputs[?OutputKey==`TaskDefinitionPython`].OutputValue')

TASKDEF_NODE=$(aws cloudformation describe-stacks \
    --stack-name zim \
    --output text \
    --query 'Stacks[0].Outputs[?OutputKey==`TaskDefinitionNode`].OutputValue')

TASKDEF_HASKELL=$(aws cloudformation describe-stacks \
    --stack-name zim \
    --output text \
    --query 'Stacks[0].Outputs[?OutputKey==`TaskDefinitionHaskell`].OutputValue')

SG=$(aws cloudformation describe-stacks \
    --stack-name zim \
    --output text \
    --query 'Stacks[0].Outputs[?OutputKey==`FargateContainerSecurityGroup`].OutputValue')

SUBNET1=$(aws cloudformation describe-stacks \
    --stack-name zim \
    --output text \
    --query 'Stacks[0].Outputs[?OutputKey==`PrivateSubnetOne`].OutputValue')

SUBNET2=$(aws cloudformation describe-stacks \
    --stack-name zim \
    --output text \
    --query 'Stacks[0].Outputs[?OutputKey==`PrivateSubnetTwo`].OutputValue')

BUCKET=$(aws cloudformation describe-stacks \
    --stack-name zim \
    --output text \
    --query 'Stacks[0].Outputs[?OutputKey==`BucketName`].OutputValue')

QUEUE_GO=$(aws cloudformation describe-stacks \
    --stack-name zim \
    --output text \
    --query 'Stacks[0].Outputs[?OutputKey==`MessageQueueGo`].OutputValue')

QUEUE_PYTHON=$(aws cloudformation describe-stacks \
    --stack-name zim \
    --output text \
    --query 'Stacks[0].Outputs[?OutputKey==`MessageQueuePython`].OutputValue')

QUEUE_NODE=$(aws cloudformation describe-stacks \
    --stack-name zim \
    --output text \
    --query 'Stacks[0].Outputs[?OutputKey==`MessageQueueNode`].OutputValue')

QUEUE_HASKELL=$(aws cloudformation describe-stacks \
    --stack-name zim \
    --output text \
    --query 'Stacks[0].Outputs[?OutputKey==`MessageQueueHaskell`].OutputValue')

ATHENS_LB=$(aws cloudformation describe-stacks \
    --stack-name zim \
    --output text \
    --query 'Stacks[0].Outputs[?OutputKey==`AthensLoadBalancerName`].OutputValue')

echo "{\"cluster\":\"$CLUSTER\",\"task_definitions\":{\"go\":\"$TASKDEF_GO\",\"python\":\"$TASKDEF_PYTHON\",\"node\":\"$TASKDEF_NODE\",\"haskell\":\"$TASKDEF_HASKELL\"},\"security_group\":\"$SG\",\"subnets\":[\"$SUBNET1\",\"$SUBNET2\"],\"bucket\":\"$BUCKET\",\"queues\":{\"go\":\"$QUEUE_GO\",\"python\":\"$QUEUE_PYTHON\",\"node\":\"$QUEUE_NODE\",\"haskell\":\"$QUEUE_HASKELL\"},\"athens\":\"$ATHENS_LB\"}"
