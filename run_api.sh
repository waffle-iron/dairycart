docker system prune -f
docker-compose --file api/docker-compose.yml up --build --remove-orphans --force-recreate --abort-on-container-exit