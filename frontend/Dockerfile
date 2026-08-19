# --- build stage ---
FROM node:20-alpine AS build
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install
COPY . .
# VITE_API_URL is baked in at build time (Vite inlines env at build).
ARG VITE_API_URL=http://localhost:8080
ENV VITE_API_URL=$VITE_API_URL
RUN npm run build

# --- serve stage ---
FROM nginx:1.27-alpine
# nginx.conf is a template: the base image's entrypoint runs envsubst on files
# in /etc/nginx/templates/ at startup. ENVSUBST_FILTER=PORT so only ${PORT} is
# substituted (nginx's own $uri etc. are left intact). PORT defaults to 80 and
# is overridden at runtime by the platform (e.g. Railway).
COPY nginx.conf /etc/nginx/templates/default.conf.template
COPY --from=build /app/dist /usr/share/nginx/html
ENV PORT=80
ENV NGINX_ENVSUBST_FILTER=PORT
EXPOSE 80
