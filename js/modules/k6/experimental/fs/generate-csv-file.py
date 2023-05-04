#!/usr/bin/env python3

import os
import random
import string

def random_string(length):
    return ''.join(random.choice(string.ascii_letters) for _ in range(length))

def generate_user_line():
    user_id = random.randint(1, 100000)
    first_name = random_string(random.randint(3, 10))
    last_name = random_string(random.randint(3, 10))
    age = random.randint(18, 99)
    email = f"{first_name}.{last_name}@example.com"

    return f"{user_id},{first_name},{last_name},{age},{email}\n"

def generate_data_csv(target_file_size):
    current_size = 0
    with open('data.csv', 'w') as f:
        while current_size < target_file_size:
            user_line = generate_user_line()
            f.write(user_line)
            current_size = f.tell()

if __name__ == "__main__":
    target_file_size = 100 * 1024 * 1024  # 100 MB
    # target_file_size = 10 * 1024 * 1024 # 10 MB
    # target_file_size = 1 * 1024 * 1024 # 1 MB
    # target_file_size = 10 * 1024 # 10 KB
    # target_file_size = 500 * 1024 # 500 KB
    generate_data_csv(target_file_size)
