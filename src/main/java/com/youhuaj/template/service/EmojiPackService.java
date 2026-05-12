package com.yupi.template.service;

import cn.hutool.core.util.StrUtil;
import com.yupi.template.config.EmojiPackConfig;
import com.yupi.template.model.enums.ImageMethodEnum;
import lombok.extern.slf4j.Slf4j;
import org.jsoup.Jsoup;
import org.jsoup.nodes.Document;
import org.jsoup.nodes.Element;
import org.jsoup.select.Elements;
import org.springframework.stereotype.Service;

import jakarta.annotation.Resource;
import java.net.URLEncoder;
import java.nio.charset.StandardCharsets;

import static com.yupi.template.constant.ArticleConstant.PICSUM_URL_TEMPLATE;

/**
 * 表情包检索服务（基于 Bing 图片搜索）
 * 程序自动在关键词后拼接"表情包"进行搜索
 *
 * @author <a href="https://codefather.cn">编程导航学习圈</a>
 */
@Service
@Slf4j
public class EmojiPackService implements ImageSearchService {

    @Resource
    private EmojiPackConfig emojiPackConfig;

    @Override
    public String searchImage(String keywords) {
        if (StrUtil.isBlank(keywords)) {
            log.warn("表情包搜索关键词为空");
            return null;
        }

        try {
            // 1. 构建搜索词（程序固定拼接"表情包"）
            String searchText = keywords + emojiPackConfig.getSuffix();
            log.info("表情包搜索: {} -> {}", keywords, searchText);

            // 2. 构建搜索 URL
            String fetchUrl = buildSearchUrl(searchText);

            // 3. 使用 Jsoup 获取页面
            Document document = Jsoup.connect(fetchUrl)
                    .timeout(emojiPackConfig.getTimeout())
                    .userAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
                    .get();

            // 4. 定位图片容器
            Element div = document.getElementsByClass("dgControl").first();
            if (div == null) {
                log.warn("Bing 未找到图片容器, keywords={}", keywords);
                return null;
            }

            // 5. 使用 CSS 选择器提取图片
            Elements imgElements = div.select("img.mimg");
            if (imgElements.isEmpty()) {
                log.warn("Bing 未检索到表情包, keywords={}, searchText={}", keywords, searchText);
                return null;
            }

            // 6. 获取第一张图片 URL
            String imageUrl = imgElements.get(0).attr("src");
            if (StrUtil.isBlank(imageUrl)) {
                log.warn("图片 URL 为空, keywords={}", keywords);
                return null;
            }

            // 7. 清理 URL 参数（移除 ?w=xxx&h=xxx）
            imageUrl = cleanImageUrl(imageUrl);

            log.info("表情包检索成功: {} -> {}", keywords, imageUrl);
            return imageUrl;

        } catch (Exception e) {
            log.error("表情包检索异常, keywords={}", keywords, e);
            return null;
        }
    }

    @Override
    public ImageMethodEnum getMethod() {
        return ImageMethodEnum.EMOJI_PACK;
    }

    @Override
    public String getFallbackImage(int position) {
        return String.format(PICSUM_URL_TEMPLATE, position);
    }

    /**
     * 构建 Bing 图片搜索 URL
     *
     * @param searchText 搜索文本
     * @return 完整的搜索 URL
     */
    private String buildSearchUrl(String searchText) {
        String encodedText = URLEncoder.encode(searchText, StandardCharsets.UTF_8);
        // 必须添加 mmasync=1 参数
        return String.format("%s?q=%s&mmasync=1", 
                emojiPackConfig.getSearchUrl(), 
                encodedText);
    }

    /**
     * 清理图片 URL 参数
     * 移除 ?w=xxx&h=xxx 等参数，避免图片质量下降和访问问题
     *
     * @param url 原始图片 URL
     * @return 清理后的 URL
     */
    private String cleanImageUrl(String url) {
        if (StrUtil.isBlank(url)) {
            return url;
        }
        
        int questionMarkIndex = url.indexOf("?");
        if (questionMarkIndex > 0) {
            return url.substring(0, questionMarkIndex);
        }
        
        return url;
    }
}
