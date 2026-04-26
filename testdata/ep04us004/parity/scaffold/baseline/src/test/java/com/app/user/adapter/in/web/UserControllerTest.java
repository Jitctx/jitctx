package com.app.user.adapter.in.web;

import com.app.user.port.in.CreateUserUseCase;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

@ExtendWith(MockitoExtension.class)
public class UserControllerTest {

    @Mock
    private CreateUserUseCase createUserUseCase;

    @InjectMocks
    private UserController userController;


    @Test
    void postUsers_shouldDoSomething() {
        // TODO: implement test
    }
}
