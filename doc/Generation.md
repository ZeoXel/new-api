The overall endpoint is:

POST https://api.tripo3d.ai/v2/openapi/task
Image to Model
Request
type: This field must be set to image_to_model.

model_version (Optional): Model version. Available versions are as below:

Turbo-v1.0-20250506
v3.0-20250812
v2.5-20250123
v2.0-20240919
v1.4-20240625
v1.3-20240522 (Deprecated)
If this option is not set, the default value will be v2.5-20250123.

file: Specifies the image input. The resolution of each image must be between 20 x 20px and 6000 x 6000px. The suggested resolution should be more than 256 x 256px

type: Indicates the file type. Although currently not validated, specifying the correct file type is strongly advised.
file_token: The identifier you get from upload, please read Docs/Upload. Mutually exclusive with url and object.
url: A direct URL to the image. Supports JPEG and PNG formats with maximum size of 20MB. Mutually exclusive with file_token and object.
object: The information you get from upload (STS), please read Docs/Upload (STS). Mutually exclusive with url and file_token.
bucket: Normally should be tripo-data.
key: The resource_uri got from session token.
model_seed (Optional): This is the random seed for model generation. For model_version>=v2.0-20240919, the seed controls the geometry generation process, ensuring identical models when the same seed is used. This parameter is an integer and is randomly chosen if not set.

The options below are only valid for model_version>=v2.0-20240919

face_limit (Optional): Limits the number of faces on the output model. If this option is not set, the face limit will be adaptively determined. If smart_low_poly=true, it should be 1000~16000, if quad=true as well, it should be 500~8000.
texture (Optional): A boolean option to enable texturing. The default value is true, set false to get a base model without any textures.
pbr (Optional): A boolean option to enable pbr. The default value is true, set false to get a model without pbr. If this option is set to true, texture will be ignored and used as true.
texture_seed (Optional): This is the random seed for texture generation for model_version>=v2.0-20240919. Using the same seed will produce identical textures. This parameter is an integer and is randomly chosen if not set. If you want a model with different textures, please use same model_seed and different texture_seed.
texture_alignment (Optional): Determines the prioritization of texture alignment in the 3D model. The default value is original_image.
original_image: Prioritizes visual fidelity to the source image. This option produces textures that closely resemble the original image but may result in minor 3D inconsistencies.
geometry: Prioritizes 3D structural accuracy. This option ensures better alignment with the model’s geometry but may cause slight deviations from the original image appearance.
texture_quality (Optional): This parameter controls the texture quality. detailed provides high-resolution textures, resulting in more refined and realistic representation of intricate parts. This option is ideal for models where fine details are crucial for visual fidelity. The default value is standard.
auto_size (Optional): Automatically scale the model to real-world dimensions, with the unit in meters. The default value is false.
style (Optional): Defines the artistic style or transformation to be applied to the 3D model, altering its appearance according to preset options. Omit this option to keep the original style and appearance.
orientation (Optional): Set orientation=align_image to automatically rotate the model to align the original image. The default value is default.
quad (Optional): Set true to enable quad mesh output. If quad=true and face_limit is not set, the default face_limit will be 10000.
Note: Enabling this option will force the output to be an FBX model.
compress (Optional): Specifies the compression type to apply to the texture. Available values are:
geometry: Applies geometry-based compression to optimize the output, By Default we use meshopt compression
smart_low_poly (Optional): Generate low-poly meshes with hand‑crafted topology. Inputs with less complexity work best. There is a possibility of failure for complex models. The default value is false.
generate_parts (Optional): Generate segmented 3D models and make each part editable. The default value is false.
Note: generate_parts is not compatible with texture=true or pbr=true, if you want to generate parts, please set texture=false and pbr=false; generate_parts is not compatible with quad=true, if you want to generate parts, please set quad=false.
The options below are only valid for model_version>=v3.0-20250812

geometry_quality (Optional):
Ultra Mode: Maximum detail for the most intricate and realistic models when setting to detailed
Standard Mode: Balanced detail and speed. The default value is standard
Style Types	Description	Output
person:person2cartoon	Transforms the model into a cartoon-style version of input character.	
object:clay	Applies a clay-like appearance to the object.	
object:steampunk	Applies a steampunk aesthetic with metallic gears and vintage details.	
animal:venom	Applies a venom-like, dark, and glossy appearance to the animal model, BTW, very horrible.	
object:barbie	Applies a barbie style to the object.	
object:christmas	Applies a christmas style to the object.	
gold	Applies a gold style to the object.	
ancient_bronze	Applies a ancient bronze style to the object.	
Response
task_id: The identifier for the successfully submitted task.
Behaviour
You can expect the same behaviour with text to model.

Example
Request:

Library:
curl
curl -X POST 'https://api.tripo3d.ai/v2/openapi/task' \
-H 'Content-Type: application/json' \
-H "Authorization: Bearer ${APIKEY}" \
-d '{        "type": "image_to_model",
             "file": {
                       "type": "jpg",
                       "file_token": "***"
                     }
    }'
Response:

{
  "code": 0,
  "data": {
    "task_id": "1ec04ced-4b87-44f6-a296-beee80777941"
  }
}
Errors
HTTP Status Code	Error Code	Description	Suggestion
429	2000	You have exceeded the limit of generation.	Please retry later.
For more infomation, please refer to Generation Rate Limit.
400	2002	The task type is unsupported.	Please check if you passed the correct task type.
400	2003	The input file is empty.	Please check if you passed file, or it’s may rejected by our firewall.
400	2004	The file type is unsupported.	Please check if the file you input is supported.
400	2008	Task is rejected because the input violates our content policy.	Please modify your input and retry.
If you believe the input should be valid, please contact us.
403	2010	You need more credits to start a new task.	Please reivew your usage at Billing and purchase more credits.
400	2015	The version has been deprecated, please try higher version.	Try higher version please.
400	2018	The model is too complex to remesh	Try another model
Text to Image
Request
type: Must be set to text_to_image.
prompt: Text input that directs the model generation.
The maximum prompt length is 1024 characters, equivalent to approximately 100 words.
The API supports multiple languages. However, emojis and certain special Unicode characters are not supported.
negative_prompt (Optional): Unlike prompt, it provides a reverse direction to assist in generating content contrasting with the original prompt. The maximum length is 255 characters.
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
-d '{"type": "text_to_image", "prompt": "a small cat"}'
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
400	2008	Task is rejected because the input violates our content policy.	Please modify your input and retry.
If you believe the input should be valid, please contact us.
403	2010	You need more credits to start a new task.	Please reivew your usage at Billing and purchase more credits.
400	2015	The version has been deprecated, please try higher version.	Try higher version please.
Advanced Generate Image
The improved image generation task type, which supports image prompt and more detailed parameters.

Request
type: Must be set to generate_image.

model_version (Optional): Model version. Available versions are as below:

flux.1_kontext_pro (default)
flux.1_dev (unable to use with image file)
gpt_4o
gemini_2.5_flash_image_preview (also known as nano banana)
prompt: Text input that directs the model generation.

The maximum prompt length is 1024 characters, equivalent to approximately 100 words.
The API supports multiple languages. However, emojis and certain special Unicode characters are not supported.
file (optional): Specifies the image input. The resolution of image must be between 20px and 6000px. The suggested resolution should be more than 256px.

type: Indicates the file type. Although currently not validated, specifying the correct file type is strongly advised.
file_token: The identifier you get from upload, please read Docs/Upload. Mutually exclusive with url and object.
url: A direct URL to the image. Supports JPEG and PNG formats with maximum size of 20MB. Mutually exclusive with file_token and object.
object: The information you get from upload (STS), please read Docs/Upload (STS). Mutually exclusive with url and file_token.
bucket: Normally should be tripo-data.
key: The resource_uri got from session token.
t_pose (optional): A bool value to transform your object to t pose while keeping main characteristics. The default value is false.

sketch_to_render (optional): A bool value to transform your sketch to a rendered image. The default value is false.

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
-d '{"type": "generate_image", "prompt": "a small cat"}'
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
400	2008	Task is rejected because the input violates our content policy.	Please modify your input and retry.
If you believe the input should be valid, please contact us.
403	2010	You need more credits to start a new task.	Please reivew your usage at Billing and purchase more credits.
400	2015	The version has been deprecated, please try higher version.	Try higher version please.
Text to Model
Request
type: Must be set to text_to_model.

model_version (Optional): Model version. Available versions are as below:

Turbo-v1.0-20250506
v3.0-20250812
v2.5-20250123
v2.0-20240919
v1.4-20240625
v1.3-20240522 (Deprecated)
If this option is not set, the default value will be v2.5-20250123.

prompt: Text input that directs the model generation.

The maximum prompt length is 1024 characters, equivalent to approximately 100 words.
The API supports multiple languages. However, emojis and certain special Unicode characters are not supported.
negative_prompt (Optional): Unlike prompt, it provides a reverse direction to assist in generating content contrasting with the original prompt. The maximum length is 255 characters.

image_seed (Optional): This is the random seed used for the process based on the prompt. This parameter is an integer and is randomly chosen if not set.

model_seed (Optional): This is the random seed for model generation. For model_version>=v2.0-20240919, the seed controls the geometry generation process, ensuring identical models when the same seed is used. This parameter is an integer and is randomly chosen if not set.

The options below are only valid for model_version>=v2.0-20240919

face_limit (Optional): Limits the number of faces on the output model. If this option is not set, the face limit will be adaptively determined. If smart_low_poly=true, it should be 1000~16000, if quad=true as well, it should be 500~8000.
texture : A boolean option to enable texturing. The default value is true, set false to get a base model without any textures.
pbr (Optional): A boolean option to enable pbr. The default value is true, set false to get a model without pbr. If this option is set to true, texture will be ignored and used as true.
texture_seed (Optional): This is the random seed for texture generation for model_version>=v2.0-20240919. Using the same seed will produce identical textures. This parameter is an integer and is randomly chosen if not set. If you want a model with different textures, please use same model_seed and different texture_seed.
texture_quality (Optional): This parameter controls the texture quality. detailed provides high-resolution textures, resulting in more refined and realistic representation of intricate parts. This option is ideal for models where fine details are crucial for visual fidelity. The default value is standard.
auto_size (Optional): Automatically scale the model to real-world dimensions, with the unit in meters. The default value is false.
style (Optional): Defines the artistic style or transformation to be applied to the 3D model, altering its appearance according to preset options. Omit this option to keep the original style and apperance.
quad (Optional): Set true to enable quad mesh output. If quad=true and face_limit is not set, the default face_limit will be 10000.
Note: Enabling this option will force the output to be an FBX model.
compress (Optional): Specifies the compression type to apply to the texture. Available values are:
geometry: Applies geometry-based compression to optimize the output, By default we use meshopt compression .
smart_low_poly (Optional): Generate low-poly meshes with hand‑crafted topology. Inputs with less complexity work best. There is a possibility of failure for complex models. The default value is false.
generate_parts (Optional): Generate segmented 3D models and make each part editable. The default value is false.
Note: generate_parts is not compatible with texture=true or pbr=true, if you want to generate parts, please set texture=false and pbr=false; generate_parts is not compatible with quad=true, if you want to generate parts, please set quad=false.
The options below are only valid for model_version>=v3.0-20250812

geometry_quality (Optional):
Ultra Mode: Maximum detail for the most intricate and realistic models when setting to detailed
Standard Mode: Balanced detail and speed. The default value is standard
Response
task_id: The identifier for the successfully submitted task.
Behaviour
Once the task moves out of the waiting queue, it typically completes within a few seconds.

Below are options you can use to customize the behavior and appearance of models in your prompts.

Specifying Poses
To set your model in a specific pose, you can append T-pose or A-pose to the end of your prompt. For example, godzilla, A-pose will position Godzilla in an A-pose.

For more detailed customization, you can specify additional ratios for the T-pose using the format T-pose:A:B:C:D:E, where:

A: Head-to-body height ratio,
B: Head-to-body width ratio,
C: Legs-to-body height ratio,
D: Arms-to-body length ratio,
E: Span of two legs, range from 0 to 15, default 9 if not specified.
Using T-pose without additional parameters implies the default ratios: T-pose:1:1:1:1:9.

The parameters are now “relative” values, e.g., to increase the leg length, you may try something like T/A-pose:1:1:1.05:1

Please ensure your input matches the specified format exactly. For instance, the following examples will not be recognized:

Incorrect: “godzilla, Apose” — Missing hyphen.
Incorrect: “godzilla, a-pose” — The letter “A” must be uppercase.
Incorrect: “godzilla, T-pose:1:1.2:1” — Incomplete arguments.
This format allows precise control over the model’s posture, enhancing the specificity and quality of your generated content.

Example
Request:

Library:
curl
export APIKEY="tsk_***"
curl https://api.tripo3d.ai/v2/openapi/task \
-H 'Content-Type: application/json' \
-H "Authorization: Bearer ${APIKEY}" \
-d '{"type": "text_to_model", "prompt": "a small cat"}'
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
400	2008	Task is rejected because the input violates our content policy.	Please modify your input and retry.
If you believe the input should be valid, please contact us.
403	2010	You need more credits to start a new task.	Please reivew your usage at Billing and purchase more credits.
400	2015	The version has been deprecated, please try higher version.	Try higher version please.
400	2018	The model is too complex to remesh	Try another model
Multiview to Model
Request
type: This field must be set to multiview_to_model.

model_version (Optional): Model version. Available versions are as below:

v3.0-20250812
v2.5-20250123
v2.0-20240919
v1.4-20240625(Deprecated)
If this option is not set, the default value will be v2.5-20250123.

files: Specifies the image inputs, this is a list contains following parameters. The list must contain exactly 4 items in the order [front, left, back, right]. You may omit certain input files by omitting the file_token, but the front input cannot be omitted. Do not use less than two images to generate. The resolution of each image must be between 20 x 20px and 6000 x 6000px. The suggested resolution should be more than 256 x 256px

type: Indicates the file type. Although currently not validated, specifying the correct file type is strongly advised.
file_token: The identifier you get from upload, please read Docs/Upload. Mutually exclusive with url and object.
url: A direct URL to the image. Supports JPEG and PNG formats with maximum size of 20MB. Mutually exclusive with file_token and object.
object: The information you get from upload (STS), please read Docs/Upload (STS). Mutually exclusive with url and file_token.
bucket: Normally should be tripo-data.
key: The resource_uri got from session token.
face_limit (Optional): Limits the number of faces on the output model. If this option is not set, the face limit will be adaptively determined. If smart_low_poly=true, it should be 1000~16000, if quad=true as well, it should be 500~8000.

texture (Optional): A boolean option to enable texturing. The default value is true, set false to get a base model without any textures.

pbr (Optional): A boolean option to enable pbr. The default value is true, set false to get a model without pbr. If this option is set to true, texture will be ignored and used as true.

texture_seed (Optional): This is the random seed for texture generation. Using the same seed will produce identical textures. This parameter is an integer and is randomly chosen if not set. If you want a model with different textures, please use same model_seed and different texture_seed.

texture_alignment (Optional): Determines the prioritization of texture alignment in the 3D model. The default value is original_image.

original_image: Prioritizes visual fidelity to the source image. This option produces textures that closely resemble the original image but may result in minor 3D inconsistencies.
geometry: Prioritizes 3D structural accuracy. This option ensures better alignment with the model’s geometry but may cause slight deviations from the original image appearance.
texture_quality (Optional): This parameter controls the texture quality. detailed provides high-resolution textures, resulting in more refined and realistic representation of intricate parts. This option is ideal for models where fine details are crucial for visual fidelity. The default value is standard.

auto_size (Optional): Automatically scale the model to real-world dimensions, with the unit in meters. The default value is false.

orientation (Optional): Set orientation=align_image to automatically rotate the model to align the original image. The default value is default.

quad (Optional): Set true to enable quad mesh output. If quad=true and face_limit is not set, the default face_limit will be 10000.

Note: Enabling this option will force the output to be an FBX model.
compress (Optional): Specifies the compression type to apply to the texture. Available values are:

geometry: Applies geometry-based compression to optimize the output, By default we use meshopt compression.
smart_low_poly (Optional): Generate low-poly meshes with hand‑crafted topology. Inputs with less complexity work best. There is a possibility of failure for complex models. The default value is false.

generate_parts (Optional): Generate segmented 3D models and make each part editable. The default value is false.

Note: generate_parts is not compatible with texture=true or pbr=true, if you want to generate parts, please set texture=false and pbr=false; generate_parts is not compatible with quad=true, if you want to generate parts, please set quad=false.
Note: The directions of object in images should be [0°, 90°, 180°, 270°] and the object should be consistent among these images. left means the left arm of the input character for example.

The options below are only valid for model_version>=v3.0-20250812

geometry_quality (Optional):
Ultra Mode: Maximum detail for the most intricate and realistic models when setting to detailed
Standard Mode: Balanced detail and speed. The default value is standard
front	left	back	right
			
Response
task_id: The identifier for the successfully submitted task.
Behaviour
You can expect the same behaviour with text to model.

Example
Request:

Library:
curl
curl -X POST 'https://api.tripo3d.ai/v2/openapi/task' \
-H 'Content-Type: application/json' \
-H "Authorization: Bearer ${APIKEY}" \
-d '{        "type": "multiview_to_model",
             "files": [
                     {
                       "type": "jpg",
                       "file_token": "***"
                     },
                     {},
                     {
                       "type": "jpg",
                       "file_token": "***"
                     },
                     {
                       "type": "jpeg",
                       "url": "***"
                     }
              ],
              "model_version": "v2.5-20250123"
    }'
Response:

{
  "code": 0,
  "data": {
    "task_id": "1ec04ced-4b87-44f6-a296-beee80777941"
  }
}
Errors
HTTP Status Code	Error Code	Description	Suggestion
429	2000	You have exceeded the limit of generation.	Please retry later.
For more infomation, please refer to Generation Rate Limit.
400	2002	The task type is unsupported.	Please check if you passed the correct task type.
400	2003	The input file is empty.	Please check if you passed file, or it’s may rejected by our firewall.
400	2004	The file type is unsupported.	Please check if the file you input is supported.
400	2008	Task is rejected because the input violates our content policy.	Please modify your input and retry.
If you believe the input should be valid, please contact us.
403	2010	You need more credits to start a new task.	Please reivew your usage at Billing and purchase more credits.
400	2015	The version has been deprecated, please try higher version.	Try higher version please.
400	2018	The model is too complex to remesh	Try another model
Texture Model
Request
type: This field must be set to texture_model.

original_model_task_id: The task_id of a previous model.

Only the task IDs of the tasks below are supported:

text_to_model

image_to_model

multiview_to_model

texture_model

The model_version of previous task should be in (Turbo-v1.0-20250506, v2.0-20240919, v2.5-20250123, v3.0-20250812).

texture_prompt:

text: Prompt text for texture image, mutually exclusive with image.
style_image (Optional): Allows you to provide a reference image to influence the artistic style of the generated model. The resolution of image must be between 20px and 6000px. The suggested resolution should be more than 256px
type: Indicates the file type. Although currently not validated, specifying the correct file type is strongly advised.
file_token: The identifier you get from upload, please read Docs/Upload. Mutually exclusive with url and object.
url: A direct URL to the image. Supports JPEG and PNG formats with maximum size of 20MB. Mutually exclusive with file_token and object.
object: The information you get from upload (STS), please read Docs/Upload (STS). Mutually exclusive with url and file_token.
bucket: Normally should be tripo-data.
key: The resource_uri got from session token.
image: Prompt image for texture image, mutually exclusive with text, This can be specified as a file token, URL, or object, following the same format as other file inputs. The resolution of image must be between 20px and 6000px. The suggested resolution should be more than 256px. This is necessary if the original task is not text_to_model, image_to_model, multiview_to_model or texture_model.
texture (Optional): A boolean option to enable texturing. The default value is true, set false to only update the pbr texture with pbr=true.

pbr (Optional): A boolean option to enable pbr. The default value is true, set false to get a model without pbr.

texture_seed (Optional): This is the random seed for texture generation. Using the same seed will produce identical textures. This parameter is an integer and is randomly chosen if not set.

texture_alignment (Optional): Determines the prioritization of texture alignment in the 3D model. The default value is original_image.

original_image: Prioritizes visual fidelity to the source image. This option produces textures that closely resemble the original image but may result in minor 3D inconsistencies.
geometry: Prioritizes 3D structural accuracy. This option ensures better alignment with the model’s geometry but may cause slight deviations from the original image appearance.
texture_quality (Optional): This parameter controls the texture quality. detailed provides high-resolution textures, resulting in more refined and realistic representation of intricate parts. This option is ideal for models where fine details are crucial for visual fidelity. The default value is standard.

part_names(Optional): The list of part names referred from Mesh Segmentation, the default value will be all part names generated from segmentation.

compress (Optional): Specifies the compression type to apply to the texture. Available values are:

geometry: Applies geometry-based compression to optimize the output, By default we use meshopt compression
model_version (Optional): Specifies the model version to use for texture generation. Available values are:

v2.5-20250123 (default)
v3.0-20250812
v2.0-20240919 (Deprecated)
Note: The version v3.0-20250812 doesn’t support multiview_to_model currently.

bake (Optional): When set to true, bakes the model’s textures, combining advanced material effects into the base textures. The default value is true.

Response
task_id: The identifier for the successfully submitted task.
Behaviour
You can expect the same behaviour with text to model.

Example
Request:

Library:
curl
curl -X POST 'https://api.tripo3d.ai/v2/openapi/task' \
-H 'Content-Type: application/json' \
-H "Authorization: Bearer ${APIKEY}" \
-d '{"type": "texture_model", "original_model_task_id": "1ec04ced-4b87-44f6-a296-beee80777941"}'
Response:

{
  "code": 0,
  "data": {
    "task_id": "e3046989-e69d-4e0d-b192-7573227e3ce5"
  }
}
Errors
HTTP Status Code	Error Code	Description	Suggestion
429	2000	You have exceeded the limit of generation.	Please retry later.
For more infomation, please refer to Generation Rate Limit.
400	2002	The task type is unsupported.	Please check if you passed the correct task type.
400	2006	The type of the input original task is invalid.	Please provide a valid task.
400	2007	The status of the original task is not success.	Use a successful original model task.
403	2010	You need more credits to start a new task.	Please reivew your usage at Billing and purchase more credits.
Refine Model
Request
type: This field must be set to refine_model.
draft_model_task_id: The task_id of a draft model. Only the task IDs of the tasks below are supported:
text_to_model
image_to_model
multiview_to_model
Note: models of model_version>=v2.0-20240919 for refine is not suppoted.

Response
task_id: The identifier for the successfully submitted task.
Behaviour
The refinement process, being considerably more complex than initial drafting, yields a lower throughput and necessitates longer wait times, which are typically about 2 minutes in addition to queueing time.

We are actively enhancing our system to improve performance, which may cause these figures to vary over time. If you require increased throughput, please contact us for further assistance.

Example
Request:

Library:
curl
curl -X POST 'https://api.tripo3d.ai/v2/openapi/task' \
-H 'Content-Type: application/json' \
-H "Authorization: Bearer ${APIKEY}" \
-d '{"type": "refine_model", "draft_model_task_id": "1ec04ced-4b87-44f6-a296-beee80777941"}'
Response:

{
  "code": 0,
  "data": {
    "task_id": "e3046989-e69d-4e0d-b192-7573227e3ce5"
  }
}
Errors
HTTP Status Code	Error Code	Description	Suggestion
429	2000	You have exceeded the limit of generation.	Please retry later.
For more infomation, please refer to Generation Rate Limit.
400	2002	The task type is unsupported.	Please check if you passed the correct task type.
400	2006	The task is not a draft task.	Please use a draft model to start refine.
It is not supported to refine a model from a non-draft model, e.g., refine a model twice.
400	2007	The status of the draft task is not success.	Use a successful draft model task to refine.
403	2010	You need more credits to start a new task.	Please reivew your usage at Billing and purchase more credits.
