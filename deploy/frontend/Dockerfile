# Simple Nginx frontend container using pre-built dist
FROM nginx:alpine

# Backend address, substituted into the nginx template at container start by
# nginx:alpine's /docker-entrypoint.d/20-envsubst-on-templates.sh. The defaults
# suit the single-host pod; override BACKEND_HOST to point nginx at another
# machine for a split deployment. NGINX_ENVSUBST_FILTER limits substitution to
# just these two names, so nginx runtime variables ($host, $scheme, ...) are
# never touched.
ENV BACKEND_HOST=127.0.0.1 \
    BACKEND_PORT=8080 \
    NGINX_ENVSUBST_FILTER=^(BACKEND_HOST|BACKEND_PORT)\$

# Copy built frontend
COPY web/dist/ /usr/share/nginx/html/

# nginx:alpine expands every /etc/nginx/templates/*.template into
# /etc/nginx/conf.d/ at startup, which the stock nginx.conf includes.
COPY deploy/nginx/default.conf.template /etc/nginx/templates/default.conf.template

# Set proper permissions (nginx user exists in base image)
RUN chown -R nginx:nginx /var/cache/nginx && \
    chown -R nginx:nginx /var/log/nginx && \
    chown -R nginx:nginx /etc/nginx/conf.d && \
    chown -R nginx:nginx /usr/share/nginx/html && \
    touch /var/run/nginx.pid && \
    chown -R nginx:nginx /var/run/nginx.pid

# Switch to non-root user
USER nginx

# Expose HTTP (and HTTPS if you add SSL later)
EXPOSE 8081 8443

CMD ["nginx", "-g", "daemon off;"]
