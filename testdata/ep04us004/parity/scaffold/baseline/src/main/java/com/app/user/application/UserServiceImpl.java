package com.app.user.application;

import com.app.user.port.in.CreateUserUseCase;
import com.app.user.port.out.UserRepository;
import org.springframework.stereotype.Service;

@Service
public class UserServiceImpl implements CreateUserUseCase {

    private final UserRepository userRepository;

    public UserServiceImpl(UserRepository userRepository) {
        this.userRepository = userRepository;
    }


    @Override
    public UserResponse execute(CreateUserCommand cmd) {
        // TODO(jitctx): implement UserServiceImpl.execute
        throw new UnsupportedOperationException("Not yet implemented");
    }
}
