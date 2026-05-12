package com.yupi.template.mapper;

import com.mybatisflex.core.BaseMapper;
import com.yupi.template.model.entity.PaymentRecord;
import org.apache.ibatis.annotations.Mapper;

/**
 * 支付记录 Mapper
 *
 * @author <a href="https://codefather.cn">编程导航学习圈</a>
 */
@Mapper
public interface PaymentRecordMapper extends BaseMapper<PaymentRecord> {
}
