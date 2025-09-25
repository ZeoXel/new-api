# 数据导入顺序

按照以下顺序导入CSV文件到Supabase，避免外键约束错误：

1. **users.csv**
2. **vendors.csv**
3. **models.csv**
4. **channels.csv**
5. **abilities.csv**
6. **tokens.csv**
7. **options.csv**
8. **redemptions.csv**
9. **prefill_groups.csv**
10. **setups.csv**
11. **two_fas.csv**
12. **two_fa_backup_codes.csv**
13. **top_ups.csv**
14. **logs.csv**
15. **midjourneys.csv**
16. **quota_data.csv**
17. **tasks.csv**

## 注意事项

- 确保已执行 `supabase_schema.sql` 创建所有表
- 导入时选择正确的表名匹配
- 如遇到错误，检查数据类型转换
