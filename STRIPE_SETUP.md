# Stripe 支付集成指南

本项目使用 Stripe 测试模式，无需企业资质，直接注册即可使用。

## 1. 注册 Stripe 账号

访问 [Stripe 官网](https://stripe.com) 注册账号（仅需邮箱）

## 2. 获取 API 密钥

1. 登录 [Stripe Dashboard](https://dashboard.stripe.com/)
2. 确保处于 **测试模式**（Test mode）
3. 进入 **开发者 -> API 密钥** 页面
4. 复制 **Secret key**（以 `sk_test_` 开头）

## 3. 配置后端

编辑 `src/main/resources/application-local.yml`：

```yaml
stripe:
  api-key: sk_test_你的密钥
  webhook-secret: whsec_你的webhook密钥  # 后续配置
  success-url: http://localhost:5173/vip?success=true
  cancel-url: http://localhost:5173/vip?cancelled=true
```

## 4. 测试支付

使用以下测试卡号：

| 卡号 | 有效期 | CVC | 说明 |
|-----|-------|-----|-----|
| 4242 4242 4242 4242 | 任意未来日期 | 任意3位数 | 支付成功 |
| 4000 0000 0000 0002 | 任意未来日期 | 任意3位数 | 支付失败 |

## 5. 配置 Webhook（本地开发）

### 方法一：使用 Stripe CLI（推荐）

1. 安装 Stripe CLI:
   ```bash
   # macOS
   brew install stripe/stripe-cli/stripe
   
   # Windows
   # 下载 https://github.com/stripe/stripe-cli/releases
   ```

2. 登录 Stripe:
   ```bash
   stripe login
   ```

3. 转发 Webhook 到本地:
   ```bash
   stripe listen --forward-to localhost:8567/webhook/stripe
   ```

4. 复制显示的 webhook 签名密钥（以 `whsec_` 开头）到配置文件

### 方法二：使用 Stripe Dashboard（生产环境）

1. 进入 **开发者 -> Webhooks** 页面
2. 点击 **添加端点**
3. 填写 URL: `https://你的域名/webhook/stripe`
4. 选择事件：
   - `checkout.session.completed`
   - `checkout.session.async_payment_succeeded`
5. 复制 **签名密钥** 到配置文件

## 6. 测试流程

1. 启动后端服务
2. 启动前端服务
3. 登录普通用户账号
4. 访问 VIP 购买页面 `/vip`
5. 点击 "立即购买"
6. 在 Stripe 支付页面使用测试卡号
7. 完成支付后，用户自动升级为 VIP

## 7. 退款功能

### 通过 API 退款

```bash
POST http://localhost:8567/payment/refund
Content-Type: application/json

{
  "reason": "用户申请退款"
}
```

### 通过 Stripe Dashboard 退款

1. 进入 **支付 -> 所有支付**
2. 找到对应的支付记录
3. 点击 **退款**

## 8. 常见问题

### Q: 如何切换到生产模式？

A: 在 Stripe Dashboard 关闭测试模式，使用生产环境的 API 密钥（以 `sk_live_` 开头）

### Q: Webhook 签名验证失败？

A: 确保 `webhook-secret` 配置正确，并且请求来自 Stripe

### Q: 支付成功但用户未升级？

A: 检查 Webhook 是否正常工作，查看后端日志

## 9. 价格说明

- **测试模式**：完全免费，不产生任何费用
- **生产模式**：Stripe 收取手续费（约 2.9% + $0.30/笔）

## 10. 安全建议

- ✅ 使用 HTTPS（生产环境）
- ✅ 验证 Webhook 签名
- ✅ 实现幂等性检查
- ✅ 定期检查支付记录
- ❌ 不要在前端暴露 Secret Key
- ❌ 不要跳过签名验证
