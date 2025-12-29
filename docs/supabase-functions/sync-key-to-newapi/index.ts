// Supabase Edge Function: sync-key-to-newapi
// 用途：接收数据库Webhook，将新密钥同步到new-api
// 部署：supabase functions deploy sync-key-to-newapi

import { serve } from "https://deno.land/std@0.168.0/http/server.ts"
import { createClient } from 'https://esm.sh/@supabase/supabase-js@2'

const corsHeaders = {
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Headers': 'authorization, x-client-info, apikey, content-type',
}

interface WebhookPayload {
  type: 'INSERT' | 'UPDATE' | 'DELETE'
  table: string
  record: {
    id: string                      // 密钥ID，如 "A000001"，对应Railway tokens.name
    key_value: string               // 密钥值，如 "sk-xxx..."
    provider: string                // 提供商，固定为 "lsapi"
    status: string                  // 状态枚举: 'active' | 'inactive'
    assigned_user_id: string | null // 关联用户UUID
    newapi_synced: boolean
    sync_error: string | null
  }
  old_record: null | Record<string, unknown>
}

serve(async (req) => {
  // 处理CORS预检请求
  if (req.method === 'OPTIONS') {
    return new Response('ok', { headers: corsHeaders })
  }

  try {
    const payload: WebhookPayload = await req.json()
    console.log('Received webhook:', JSON.stringify(payload))

    // 只处理INSERT事件
    if (payload.type !== 'INSERT' || payload.table !== 'api_keys') {
      return new Response(
        JSON.stringify({ success: true, message: 'Ignored non-insert event' }),
        { headers: { ...corsHeaders, 'Content-Type': 'application/json' } }
      )
    }

    const record = payload.record

    // 获取环境变量
    const newApiBaseUrl = Deno.env.get('NEWAPI_BASE_URL')
    const newApiAdminToken = Deno.env.get('NEWAPI_ADMIN_TOKEN')
    const supabaseUrl = Deno.env.get('SUPABASE_URL')
    const supabaseKey = Deno.env.get('SUPABASE_SERVICE_ROLE_KEY')

    if (!newApiBaseUrl || !newApiAdminToken) {
      console.error('Missing NEWAPI configuration')
      return new Response(
        JSON.stringify({ success: false, message: 'Missing configuration' }),
        { status: 500, headers: { ...corsHeaders, 'Content-Type': 'application/json' } }
      )
    }

    // 移除sk-前缀（Supabase存储sk-xxx，Railway存储xxx）
    const keyWithoutPrefix = record.key_value.replace('sk-', '')

    // 调用new-api导入接口
    // 映射关系: api_keys.id → tokens.name, api_keys.assigned_user_id → tokens.external_user_id
    const importResponse = await fetch(`${newApiBaseUrl}/api/token/import`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': newApiAdminToken,
      },
      body: JSON.stringify({
        key: keyWithoutPrefix,
        external_user_id: record.assigned_user_id || '',
        name: record.id,  // 使用api_keys.id作为tokens.name（如A000001）
        unlimited_quota: true,
      }),
    })

    const importResult = await importResponse.json()
    console.log('Import result:', JSON.stringify(importResult))

    // 更新Supabase中的同步状态
    const supabase = createClient(supabaseUrl!, supabaseKey!)

    if (importResult.success) {
      await supabase
        .from('api_keys')
        .update({
          newapi_synced: true,
          newapi_token_id: importResult.data?.token_id,
          sync_error: null,
        })
        .eq('id', record.id)

      return new Response(
        JSON.stringify({
          success: true,
          message: 'Key synced successfully',
          token_id: importResult.data?.token_id,
        }),
        { headers: { ...corsHeaders, 'Content-Type': 'application/json' } }
      )
    } else {
      await supabase
        .from('api_keys')
        .update({
          sync_error: importResult.message || 'Unknown error',
        })
        .eq('id', record.id)

      return new Response(
        JSON.stringify({
          success: false,
          message: importResult.message,
        }),
        { status: 400, headers: { ...corsHeaders, 'Content-Type': 'application/json' } }
      )
    }
  } catch (error) {
    console.error('Error:', error)
    return new Response(
      JSON.stringify({ success: false, message: error.message }),
      { status: 500, headers: { ...corsHeaders, 'Content-Type': 'application/json' } }
    )
  }
})
