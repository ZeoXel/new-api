Polling
Get a Specific Task
GET https://api.tripo3d.ai/v2/openapi/task/:task_id
Request
task_id: The task id.
Response
task_id: The unique identifier for the task, which should match the identifier used in the request.
type: Specifies the type of the task. For improved clarity and organization, consider this refinement:
status: Reflects the current status of the task, categorized into two types: finalized (indicating no further updates for this task) and ongoing. There are seven possible status values:
Ongoing:
queued: The task is awaiting its turn for processing.
running: The task is currently ongoing.
Finalized:
success: The task has been completed successfully. The output values can be used as the final result.
failed: The task has failed, often due to issues on our end. Please report this along with the task_id for support.
banned: The task has been banned due to violating our content policy.
expired: The task has expired after a certain period. Please try again and report this along with the task_id for support if it appeared again.
cancelled: The task has been cancelled.
unknown: The task’s current status cannot be determined, potentially indicating a problem at the system level. For assistance, please contact our support team with the task_id.
input: An object containing input data, the structure of which varies depending on the task_type.
output: An object containing the results of the task. This includes:
model: A URL to download the model, which by default expires after five minutes.
base_model: A URL to download the base model, which by default expires after five minutes.
pbr_model: A URL to download the pbr model, which by default expires after five minutes.
rendered_image: A URL for a preview image of the model, also with the same expiration time.
progress: A numerical indicator of the task’s progress, ranging from 0 to 100 (inclusive).
When status is queued, the progress value will be 0.
When status is running, this value indicates the task’s progress.
When status is success, the progress will be marked as 100.
In all other cases, the progress value should not be considered meaningful.
create_time: The timestamp when the task was created, providing a temporal reference.
Example
Library:
curl
export TASK_ID="ef731ad6-aeb0-4950-9a2e-2298359dfaf8"
curl https://api.tripo3d.ai/v2/openapi/task/${TASK_ID} \
  -H "Authorization: Bearer ${APIKEY}"
{
  "code": 0,
  "data": {
    "task_id": "ef731ad6-aeb0-4950-9a2e-2298359dfaf8",
    "type": "text_to_model",
    "status": "running",
    "input": {
      "prompt": "a small cat"
    },
    "output": {},
    "progress": 99,
    "create_time": 1709048933
  }
}
Behaviour
This endpoint facilitates tracking your task’s progress and accessing its output.

It is essential that the task should be accessed using the same API key that initiated the task. In other words, if you initiate a task with API key A and attempt to query it with API key B—even if both keys belong to the same user—an error indicating that the task could not be found will occur.

For real-time tracking of task generation progress, we recommend utilizing our streaming method over traditional polling for a more efficient update mechanism. Further details and examples are provided below.

Be aware that the output field might sometimes contain additional, undocumented fields. These fields can vary and may not always be available, so depending on them for essential processes is not recommended.

Errors
HTTP Status Code	Error Code	Description	Suggestion
429	2000	You have exceeded the limit of generation.	Please retry later.
For more infomation, please refer to Generation Rate Limit section below.
404	2001	Task not found.	Please check if you passed the correct task id.
