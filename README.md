# PowerGuardian
PowerGuardianは，RaspberryPi Zeroに取り付けたクランプメータで電流値を測定し，exporter形式で提供できるソフトウェアです．

## 使用方法
起動すると，`pg.properties`が生成され，`0.0.0.0:8000/metrics`でexporterが起動します．

exporterに表示される`system`や`point`は`pg.properties`で編集が可能です．

## 環境
ADS1115を使ったクランプメータ拡張基盤(ADRSZCM)を想定しています．
