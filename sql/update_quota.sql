-- 用户配额升级脚本
-- @author <a href="https://codefather.cn">编程导航学习圈</a>

use ai_passage_creator;

-- 添加 quota 字段
ALTER TABLE user ADD COLUMN quota int default 5 not null comment '剩余配额' AFTER userRole;

-- 为已有用户设置默认配额
UPDATE user SET quota = 5 WHERE quota IS NULL;
