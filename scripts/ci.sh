#!/bin/bash

if make; then
    if make integration_tests; then
        make acceptance_tests
    fi
fi
