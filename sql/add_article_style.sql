# 添加文章风格字段
# @author <a href="https://codefather.cn">编程导航学习圈</a>

use ai_passage_creator;

-- 为 article 表添加 style 字段（文章风格）
ALTER TABLE article
    ADD COLUMN style VARCHAR(20) NULL COMMENT '文章风格：tech/emotional/educational/humorous' AFTER topic;
