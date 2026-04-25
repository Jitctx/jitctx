package com.app.user.port.out;

import com.app.user.domain.User;
import java.util.Optional;

public interface UserRepository {
    Optional<User> findById(Long id);
    User save(User user);
}
