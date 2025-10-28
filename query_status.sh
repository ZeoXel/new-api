#!/bin/bash

EXECUTE_ID="$1"

if [ -z "$EXECUTE_ID" ]; then
  echo "Usage: $0 <execute_id>"
  exit 1
fi

echo "查询执行ID: $EXECUTE_ID"
echo ""

curl -s -X GET "http://localhost:3000/v1/workflows/executions/$EXECUTE_ID" \
  -H "Authorization: Bearer sk-f4S1I0MvDSnio8FbDxoPejJ6pDP5mUdSn85piIRTo8pVFC0B" \
  -H "Content-Type: application/json"

echo ""
