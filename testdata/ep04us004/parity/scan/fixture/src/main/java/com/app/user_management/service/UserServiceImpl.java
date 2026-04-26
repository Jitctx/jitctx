package com.app.user_management.service;

import com.app.user_management.port.in.CreateUserUseCase;
import com.app.user_management.port.out.UserRepository;
import com.app.user_management.domain.User;
import com.app.notification.port.in.SendNotificationUseCase;

public class UserServiceImpl implements CreateUserUseCase {
    private final UserRepository userRepository;
    private final SendNotificationUseCase sendNotification;

    public UserServiceImpl(UserRepository userRepository, SendNotificationUseCase sendNotification) {
        this.userRepository = userRepository;
        this.sendNotification = sendNotification;
    }

    public User execute(String name, String email) {
        User user = new User();
        return userRepository.save(user);
    }
}
