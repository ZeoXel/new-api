Stylization
POST https://api.tripo3d.ai/v2/openapi/task
Request
type: Must be set to stylize_model.

style: Applies unique styles to your model. It can only be following four values.

lego: Alters the model to look like Lego bricks.
voxel: Converts the model to a voxel-based form.
voronoi: Applies a Voronoi diagram effect to the model.
minecraft: Output a schem file for minecraft.
original_model_task_id: The task_id of a previous task.

Only the task IDs of the tasks below are supported:

text_to_model

image_to_model

multiview_to_model

texture_model

refine_model

animate_rig

animate_retarget

block_size(Optional): Specify the grid size. Should be ranged from 32 to 128, and the default value is 80. Currently only for minecraft.

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
	"type": "stylize_model",
	"style": "lego",
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
Conversion
POST https://api.tripo3d.ai/v2/openapi/task
Request
type: Must be set to convert_model.

format: Converts .glb format models from the OpenAPI into other formats for enhanced compatibility. It can only be following five values.

GLTF: Transforms into GLTF format.
USDZ: Transforms into USDZ format.
FBX: Transforms into FBX format.
OBJ: Transforms into OBJ format. Not supported with rigged models.
STL: Transforms into STL format. Not supported with rigged models.
3MF: Transforms into 3MF format. Not supported with rigged models and only geometry.
Note: If you need to perform a format conversion after an initial one, only the format parameter will be supported.
original_model_task_id: The task_id of a previous task.

Only the task IDs of the tasks below are supported:

text_to_model

image_to_model

multiview_to_model

texture_model

refine_model

animate_rig

animate_retarget

convert

quad (Optional): When set to true, enables quad remeshing (or auto retopology) for the converted model.

Note: GLTF and STL doesn’t support to store quad faces, so enable quad option will still do retopology and face limit, but result model will store all the faces as triangles.
force_symmetry (Optional): Effective only when quad is enabled. Force symmetry for quad remeshing.

face_limit (Optional): Limits the number of faces on the output model. The default value is 10000.

Note: If quad is enabled, the face count is based on the number of polygons; otherwise, it represents the number of triangles.
flatten_bottom (Optional): When set to true, enables flatten the bottom of converted model.

flatten_bottom_threshold (Optional): Effective only when flatten_bottom is enabled. Set the bottom flatten depth. The default value is 0.01.

texture_size (Optional): Set the diffuse color texture size (in pixel). The default value is 2048(4096 for model_version>=v2.0-20240919), and should be smaller than the default value.

texture_format (Optional): Set the diffuse color texture format (supports BMP, DPX, HDR, JPEG, OPEN_EXR, PNG, TARGA, TIFF, WEBP). The default value is JPEG(The Default value for FBX is PNG to better support unity).

pivot_to_center_bottom (Optional): Set the pivot point to center bottom. The default value is false

scale_factor (Optional): Set the object scale. The default value is 1.

with_animation (Optional): When set to true, the exported model will include skeletal binding information and animation structure. This is particularly useful for scenarios requiring further editing or animation usage in other software. The default value is true.

pack_uv (Optional): When set to true, Combine all UV islands from different parts into one unified layout and export a single texture map. The default value is false.

bake (Optional): When set to true, bakes the model’s textures, combining advanced material effects (such as normal maps, ambient occlusion, etc.) into the base textures. The default value is true.

part_names (Optional): The list of part names referred from Mesh Segmentation

animate_in_place (optional): A bool to determine if the model will be animated in fixed place. The default value is false. If already set true in retarget process, it will not affect the model on this step.

export_vertex_colors (optional): A boolean that controls whether to include vertex colors when exporting. Only can be used when OBJ or GLTF is selected in format. The default value is false.

export_orientation (optional): Set the model facing direction. The default value is +x (supports -x, -y, +y).

fbx_preset (optional, experimental): Specify the target platform for fbx export compatibility. The default value is blender (supports 3dsmax, mixamo).

Note:

Please be aware that due to some limitation, textures will not be retained in the final model when choosing STL.
If quad is enabled, the size of roughness and metallic texture map will be half of the size of base color map.
Response
task_id: The identifier for the successfully submitted task.
Behaviour
Once the task moves out of the waiting queue, it typically completes within a few seconds. If quad=true, it will take about 1 min.

Example
Request:

Library:
curl
export APIKEY="tsk_***"
curl https://api.tripo3d.ai/v2/openapi/task \
-H 'Content-Type: application/json' \
-H "Authorization: Bearer ${APIKEY}" \
-d '{
	"type": "convert_model",
	"format": "USDZ",
	"original_model_task_id": "19f4256a-80b7-4cdf-956c-dea802e4d11a",
	"quad": true,
	"face_limit": 5000
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
