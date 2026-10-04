FROM nginx:alpine

# Backend address, substituted into the nginx template at container start.
ENV BACKEND_HOST=127.0.0.1 \
    BACKEND_PORT=8080 \
    NGINX_ENVSUBST_FILTER=^(BACKEND_HOST|BACKEND_PORT)\$

# Install the config as an nginx template; nginx:alpine expands it into
# /etc/nginx/conf.d/ at startup.
COPY default.conf.template /etc/nginx/templates/default.conf.template

# Create nginx user for running as non-root
RUN addgroup -g 1000 -S nginx && \
    adduser -u 1000 -S nginx -G nginx && \
    chown -R nginx:nginx /var/cache/nginx && \
    chown -R nginx:nginx /var/log/nginx && \
    chown -R nginx:nginx /etc/nginx/conf.d && \
    touch /var/run/nginx.pid && \
    chown -R nginx:nginx /var/run/nginx.pid

# Switch to non-root user
USER nginx

EXPOSE 80 443

CMD ["nginx", "-g", "daemon off;"]
