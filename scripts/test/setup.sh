#/bin/bash

echo ">>> RESET $DIR_RES"
rm -rf "$DIR_RES"
mkdir "$DIR_RES"

echo ">>> LAUNCH SERVER"
nohup "$DIR_ROOT/run.sh" -c "$DIR_CASE/conf/server.json" 2>1 > "$DIR_RES/server.log" &

# Wait a bit for the server to come up
for WAIT in $SRV_WAIT
do
	echo -ne "."
	sleep $WAIT
done
echo "[OK]"
