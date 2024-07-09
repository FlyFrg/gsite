cd domain-list-community && go run ./ && cd ..
cd domain-list-community.ir && go run ./ && cd ..
cd domain-list-community.tm && go run ./ && cd ..
cd domain-list-community.ru && go run ./ && cd ..
cd domain-list-community.cn && go run ./ && cd ..

cp domain-list-community/dlc.dat result/dlc.dat
cp domain-list-community.ru/dlc.dat result/dlc-ru.dat
cp domain-list-community.ir/dlc.dat result/dlc-ir.dat
cp domain-list-community.tm/dlc.dat result/dlc-tm.dat
cp domain-list-community.cn/dlc.dat result/dlc-cn.dat