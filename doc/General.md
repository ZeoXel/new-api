Server URL
The current host URL is:

https://api.tripo3d.ai/v2/openapi
Request Body
Unless otherwise specified, the body of all requests should be in JSON format.

Response Structure
All endpoints adhere to a unified response structure for both success and failure scenarios, with the exception of those enabled for Server-Sent Events (SSE).

Succeeded
{
  "code": 0,
  "data": {}
}
For all succeeded requests, the code will always be 0, and the data is varied by the corresponding endpoint.

Error
{
  "code": 2002,
  "message": "This task type is not supported",
  "suggestion": "Refer to API documentation for supported task types"
}
For every request that encounters an error, the code field will be non-zero, indicating a specific error has occurred. Accompanying this, the message and suggestion fields are provided to assist in diagnosing the error and guiding you toward rectifying your code.

For all error codes and their meaning, please refer to the Error Handling page.

Trace ID
All responses, whether successful or unsuccessful, include a standard header field named X-Tripo-Trace-ID:

X-Tripo-Trace-ID: 20516a36-f45c-4454-83c8-45877b75eac8
This ID is randomly generated for each request. We recommend logging this value for tracking purposes.

In the event of an error, please provide this ID when reporting the issue to us, enabling a more effective investigation.