wget -c https://www.golangtc.com/static/go/1.9.2/go1.9.2.linux-amd64.tar.gz && sudo tar -zxf go1.9.2.linux-amd64.tar.gz -C /usr/local && rm -f go1.9.2.linux-amd64.tar.gz
#sudo tar -xf go1.9.2.linux-amd64.tar -C /usr/local

cat <<EOF >/home/vagrant/golang.sh
export GOROOT="/usr/local/go"
export PATH="/usr/local/go/bin:$PATH"
export GOPATH="/opt/gopath"
EOF

source /home/vagrant/golang.sh

# Ensure permissions are set for GOPATH
sudo chown -R vagrant:vagrant $GOPATH

cd $GOPATH/src/github.com/UranusBlockStack/uranus

make

sudo cp $GOPATH/src/github.com/UranusBlockStack/uranus/build/uranus /usr/local/bin/

cd /home/vagrant
uranus --miner_start > uranus.log 2>&1 &