Endpoint overview:

POST https://api.tripo3d.ai/v2/openapi/upload/sts
Upload Image
Request
The request body should be in multipart form and Content-Type in the HTTP Header should be multipart/form-data

file: The image you would like to upload. Accepted file types are webp,jpeg and png only. The resolution of image must be between 20px and 6000px. The suggested resolution should be more than 256px.
Response
image_token: The identifier returned for the successfully uploaded image.
Behaviour
The upload process typically completes within a few seconds.

Example
Request:

Library:
curl
export APIKEY="tsk_***"
curl -X POST 'https://api.tripo3d.ai/v2/openapi/upload/sts' \
-H 'Content-Type: multipart/form-data' \
-H "Authorization: Bearer ${APIKEY}" \
-F "file=@cat.jpeg"
unset APIKEY
Response:

{
  "code": 0,
  "data": {
    "image_token": "ce85f375-3ccc-440b-b847-571588872ec2"
  }
}
Errors
HTTP Status Code	Error Code	Description	Suggestion
429	2000	You have exceeded the limit of generation.	Please retry later.
For more infomation, please refer to Generation Rate Limit.
400	2002	The task type is unsupported.	Please check if you passed the correct task type.
400	2004	The file type is unsupported.	Please check if the file you input is supported.
