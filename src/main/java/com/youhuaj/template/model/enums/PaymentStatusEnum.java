package com.yupi.template.model.enums;

import lombok.Getter;

/**
 * 支付状态枚举
 *
 * @author <a href="https://codefather.cn">编程导航学习圈</a>
 */
@Getter
public enum PaymentStatusEnum {

    PENDING("PENDING", "待支付"),
    SUCCEEDED("SUCCEEDED", "支付成功"),
    FAILED("FAILED", "支付失败"),
    REFUNDED("REFUNDED", "已退款");

    private final String value;
    private final String description;

    PaymentStatusEnum(String value, String description) {
        this.value = value;
        this.description = description;
    }

    public static PaymentStatusEnum getByValue(String value) {
        if (value == null) {
            return null;
        }
        for (PaymentStatusEnum statusEnum : values()) {
            if (statusEnum.getValue().equals(value)) {
                return statusEnum;
            }
        }
        return null;
    }
}
