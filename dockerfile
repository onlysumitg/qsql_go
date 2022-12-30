FROM golang:1.19-bullseye
ENV DEBIAN_FRONTEND noninteractive
#RUN echo 'debconf debconf/frontend select Noninteractive' | debconf-set-selections
WORKDIR /app
COPY . ./

RUN apt-get update && \
    apt-get install -y -q --no-install-recommends dialog apt-utils && \
    apt-get install -y -q --no-install-recommends unixodbc-dev \
    unixodbc \
    libpq-dev && \
 go mod download && \
chmod +x ./drivers/ibm-iaccess-1.1.0.27-1.0.amd64.deb && \
apt-get install ./drivers/ibm-iaccess-1.1.0.27-1.0.amd64.deb && \ 
go build -o ./build/qsql ./cmd/web && \
chmod +x  ./build/qsql
 


EXPOSE 4040
CMD [ "./build/qsql" ]