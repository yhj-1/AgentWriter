package com.yupi.template.model.enums;

import lombok.Getter;

import java.util.Arrays;
import java.util.List;
import java.util.stream.Collectors;

/**
 * 文章风格枚举
 *
 * @author <a href="https://codefather.cn">编程导航学习圈</a>
 */
@Getter
public enum ArticleStyleEnum {

    TECH("tech", "科技风格"),
    EMOTIONAL("emotional", "情感风格"),
    EDUCATIONAL("educational", "教育风格"),
    HUMOROUS("humorous", "轻松幽默风格");

    private final String value;
    private final String text;

    ArticleStyleEnum(String value, String text) {
        this.value = value;
        this.text = text;
    }

    /**
     * 获取所有值列表
     */
    public static List<String> getValues() {
        return Arrays.stream(values())
                .map(ArticleStyleEnum::getValue)
                .collect(Collectors.toList());
    }

    /**
     * 根据 value 获取枚举
     */
    public static ArticleStyleEnum getEnumByValue(String value) {
        if (value == null || value.isEmpty()) {
            return null;
        }
        for (ArticleStyleEnum styleEnum : ArticleStyleEnum.values()) {
            if (styleEnum.getValue().equals(value)) {
                return styleEnum;
            }
        }
        return null;
    }

    /**
     * 校验是否为有效的风格值
     */
    public static boolean isValid(String value) {
        if (value == null || value.isEmpty()) {
            return true; // 允许为空
        }
        return getEnumByValue(value) != null;
    }
}
