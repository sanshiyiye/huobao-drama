-- 为 characters 表添加 age、gender 字段
-- 创建时间: 2026-02-21
-- 说明: 支持角色设计弹框中的年龄、性别设定（提取时可选输出，弹框可编辑）

ALTER TABLE characters ADD COLUMN age VARCHAR(20);
ALTER TABLE characters ADD COLUMN gender VARCHAR(10);
