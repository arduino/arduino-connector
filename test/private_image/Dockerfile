# push on private arduino repo with
#docker build -t test-sleeper . &&\
#docker tag test-sleeper:latest <your private reg>/test-sleeper:latest &&\
#docker push <your private reg>/test-sleeper:latest

FROM centos
ADD run.sh /tmp/run.sh
RUN chmod +x /tmp/run.sh
ENTRYPOINT ["/tmp/run.sh"]