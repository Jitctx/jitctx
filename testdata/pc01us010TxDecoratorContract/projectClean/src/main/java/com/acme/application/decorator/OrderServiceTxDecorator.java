package com.acme.application.decorator;

import org.springframework.context.annotation.Primary;
import org.springframework.beans.factory.annotation.Qualifier;

@Primary
@Qualifier("txDecorator")
public class OrderServiceTxDecorator {
}
