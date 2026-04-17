package com.app.notification.port.in;

public interface SendNotificationUseCase {
    void send(String recipient, String message);
}
