package com.app.user.service;

import com.app.user.domain.User;
import com.app.user.port.in.CreateUserUseCase;
import com.app.user.port.out.UserRepository;
import org.springframework.stereotype.Service;

@Service
public class CreateUserService implements CreateUserUseCase {
    private final UserRepository userRepository;

    public CreateUserService(UserRepository userRepository) {
        this.userRepository = userRepository;
    }

    @Override
    public User execute(String name, String email) {
        return userRepository.save(new User());
    }
}
