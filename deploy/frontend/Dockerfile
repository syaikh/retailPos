# Simple Nginx frontend container using pre-built dist
FROM nginx:alpine

# Copy built frontend
COPY web/dist/ /usr/share/nginx/html/

# Copy custom nginx config
COPY deploy/nginx/nginx.conf /etc/nginx/nginx.conf

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
