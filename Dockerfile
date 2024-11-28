# Use the official nginx image as base
FROM nginx:alpine

# Create index.html directly in the container
RUN echo '<!DOCTYPE html>\
<html>\
<head>\
    <title>Welcome to pipe3</title>\
</head>\
<body>\
    <div class="container">\
        <h1>Welcome to pipe!</h1>\
        <p>If you see this page, the nginx web server is successfully installed and working.</p>\
        <p>This page was created from within the Dockerfile.</p>\
    </div>\
</body>\
</html>' > /usr/share/nginx/html/index.html

# Expose port 80
EXPOSE 80

# Start nginx in the foreground
CMD ["nginx", "-g", "daemon off;"]
