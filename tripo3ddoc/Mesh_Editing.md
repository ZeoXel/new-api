Mesh Segmentation
POST https://api.tripo3d.ai/v2/openapi/task
Request
type: Must be set to mesh_segmentation.

model_version (Optional): Model version. Available versions are as below:

v1.0-20250506
original_model_task_id: The task_id of a previous task.

Only the task IDs of the tasks below are supported:

text_to_model

image_to_model

multiview_to_model

texture_model

refine_model

highpoly_to_lowpoly

Response
task_id: The identifier for the successfully submitted task.
Behaviour
Once the task moves out of the waiting queue, it typically completes within several minutes. After this operation, it will separate the whole model in parts based on the type and shape of it. After segmentation and importing into 3D editing software, like Blender, you will see part names in collection area.

Example
Request:

Library:
curl
export APIKEY="tsk_***"
curl https://api.tripo3d.ai/v2/openapi/task \
-H 'Content-Type: application/json' \
-H "Authorization: Bearer ${APIKEY}" \
-d '{
	"type": "mesh_segmentation",
	"original_model_task_id": "19f4256a-80b7-4cdf-956c-dea802e4d11a",
}'
unset APIKEY
Response:

{
  "code": 0,
  "data": {
    "task_id": "1ec04ced-4b87-44f6-a296-beee80777941"
  }
}
Mesh Completion
POST https://api.tripo3d.ai/v2/openapi/task
Request
type: Must be set to mesh_completion.

model_version (Optional): Model version. Available versions are as below:

v1.0-20250506
part_names(Optional): The list of part names referred from Mesh Segmentation, the default value will be all part names generated from segmentation.

original_model_task_id: The task_id of a previous task.

Only the task IDs of the tasks below are supported:

mesh_segmentation
Response
task_id: The identifier for the successfully submitted task.
Behaviour
Once the task moves out of the waiting queue, it typically completes within a few seconds.

Example
Request:

Library:
curl
export APIKEY="tsk_***"
curl https://api.tripo3d.ai/v2/openapi/task \
-H 'Content-Type: application/json' \
-H "Authorization: Bearer ${APIKEY}" \
-d '{
	"type": "mesh_completion",
	"original_model_task_id": "19f4256a-80b7-4cdf-956c-dea802e4d11a",
}'
unset APIKEY
Response:

{
  "code": 0,
  "data": {
    "task_id": "1ec04ced-4b87-44f6-a296-beee80777941"
  }
}
Smart LowPoly
POST https://api.tripo3d.ai/v2/openapi/task
Request
type: Must be set to highpoly_to_lowpoly.

original_model_task_id: The task_id of a previous task.

Only the task IDs of the tasks below are supported:

text_to_model

image_to_model

multiview_to_model

texture_model

refine_model

mesh_segmentation

mesh_completion

model_version (Optional): Model version. Available versions are as below:

P-v1.0-20250506
quad: Determined if the final model generated in quad or triangle face, the default value is false.

part_names(optional): The list of part names referred from Mesh Segmentation, the default value is empty.

face_limit(optional): Determined the amount of face for model when generation, range from 1000 to 16000.

bake(optional): When set to true, the model will be baked when generation, the default value is true.

Response
task_id: The identifier for the successfully submitted task.
Behaviour
Once the task moves out of the waiting queue, it typically completes within a few seconds.

Example
Request:

Library:
curl
export APIKEY="tsk_***"
curl https://api.tripo3d.ai/v2/openapi/task \
-H 'Content-Type: application/json' \
-H "Authorization: Bearer ${APIKEY}" \
-d '{
	"type": "highpoly_to_lowpoly",
	"original_model_task_id": "19f4256a-80b7-4cdf-956c-dea802e4d11a",
}'
unset APIKEY
Response:

{
  "code": 0,
  "data": {
    "task_id": "1ec04ced-4b87-44f6-a296-beee80777941"
  }
}
Errors
HTTP Status Code	Error Code	Description	Suggestion
429	2000	You have exceeded the limit of generation	Please retry later.
For more infomation, please refer to Generation Rate Limit.
400	2002	The task type is unsupported.	Please check if you passed the correct task type.
400	2006	The type of the input original task is invalid.	Please provide a valid task.
400	2007	The status of the original task is not success.	Use a successful original model task.
403	2010	You need more credits to start a new task.	Please reivew your usage at Billing and purchase more credits.
