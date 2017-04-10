package svm

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/icoder528/libsvm-go"
	"github.com/icoder528/ml/utils"
	"github.com/icoder528/sego"
)

//DocTraning 单个文档的特征向量数据
type DocTraning struct {
	clazz  int             //文档所属类别
	lemmas map[int]float64 //文档包含的特征词和tfidf权值
}

//LoadTraning 加载训练数据
func LoadTraning(r io.Reader) (docs []*DocTraning) {
	br := bufio.NewReader(r)
	for {
		line, err := readline(br)
		if err != nil {
			if err != io.EOF {
				log.Println(err)
			}
			break
		}

		items := strings.Split(line, " ")
		if len(items) < 2 {
			log.Println("invalid line data :", line)
			continue
		}

		label, err := strconv.Atoi(items[0])
		if err != nil {
			log.Println("invalid class lable :", items[0], " error:", err)
			continue
		}
		lemmas := map[int]float64{}
		for _, item := range items[1:] {
			item = strings.TrimSpace(item)
			if len(item) == 0 {
				continue
			}
			feature := strings.Split(item, ":")
			if len(feature) != 2 {
				log.Println("invalid feature tfidf data:", item)
				continue
			}
			index, err := strconv.Atoi(feature[0])
			if err != nil {
				log.Println("invalide feature index:", feature[0], " ", err)
				continue
			}
			tfidf, err := strconv.ParseFloat(feature[1], 64)
			if err != nil {
				log.Println("invalide feature tfidf:", feature[1], " ", err)
				continue
			}
			lemmas[index] = tfidf
		}
		docs = append(docs, &DocTraning{clazz: label, lemmas: lemmas})
	}
	return
}

func readline(r *bufio.Reader) (string, error) {
	var (
		isPrefix = true
		err      error
		line, ln []byte
	)
	for isPrefix && err == nil {
		line, isPrefix, err = r.ReadLine()
		ln = append(ln, line...)
	}
	return string(ln), err
}

//计算某个词元在文档集中的idf值
func idf(lemma int, docs []*DocTraning) float64 {
	var count float64
	for _, doc := range docs {
		if _, ok := doc.lemmas[lemma]; ok {
			count++
		}
	}
	return math.Log((float64(len(docs)) + 0.01) / (count + 0.01))
}

//Corpus 用于svm计算的数据
type Corpus struct {
	classes  map[string]int  //分类
	features map[string]int  //特征词
	transIDF map[int]float64 //训练集文档的特征词的idf
	sgter    *sego.Segmenter
}

//NewCorpus Corpus构造函数
func NewCorpus(classes, features []string, docs []*DocTraning) *Corpus {
	var (
		clsMp = map[string]int{}
		ftsMp = map[string]int{}
		idfMP = map[int]float64{}

		words []string
		sgter sego.Segmenter
	)
	for i, clazz := range classes {
		index := i + 1
		clsMp[clazz] = index
	}

	//加载特征词分词器
	for i, ft := range features {
		index := i + 1
		ftsMp[ft] = index
		idfMP[index] = idf(index, docs)
		words = append(words, ft)
	}
	sgter.LoadDictionaryReaders(sego.ModeOneKey, strings.NewReader(strings.Join(words, "\n")))

	return &Corpus{classes: clsMp, features: ftsMp, transIDF: idfMP, sgter: &sgter}
}

//NewZipCorpus 从zip中加载语料
func NewZipCorpus(mz *utils.MemZip) (*Corpus, error) {
	cr, err := mz.Get("name.map")
	if err != nil {
		return nil, err
	}
	fr, err := mz.Get("feature_mmt.txt")
	if err != nil {
		return nil, err
	}
	tr, err := mz.Get("train.date")
	if err != nil {
		return nil, err
	}

	var (
		classes, features []string
	)
	utils.TravelLines(cr, ":", func(line string, items []string) {
		if len(items) == 2 {
			classes = append(classes, strings.TrimSpace(items[0]))
		}
	})
	utils.TravelLines(fr, " ", func(line string, items []string) {
		feature := strings.TrimSpace(line)
		if len(feature) != 0 {
			features = append(features, string(feature))
		}
	})
	return NewCorpus(classes, features, LoadTraning(tr)), nil
}

//Vector 计算文本的特征向量
func (cp *Corpus) Vector(data []byte) map[int]float64 {
	vector := map[int]float64{}

	//分词统计匹配的关键词数
	count := map[int]int{}
	total := 0
	for _, sgt := range cp.sgter.Segment(data) {
		if index, ok := cp.features[sgt.Token().Text()]; ok {
			count[index]++
			total++
		}
	}
	//统计tfidf
	tfidf := map[int]float64{}
	var sum float64
	for k, v := range count {
		tf := float64(v) / float64(total)
		tfidf[k] = tf * cp.transIDF[k]
		sum += tfidf[k] * tfidf[k]
	}
	//归一化tfidf值
	scalar := math.Sqrt(sum)
	for k, v := range tfidf {
		vector[k] = v / scalar
	}

	return vector
}

//Label 根据label的索引获取label
func (cp *Corpus) Label(index int) string {
	for k, v := range cp.classes {
		if v == index {
			return k
		}
	}
	return ""
}

//Feature 根据特征词的索引获取特征词
func (cp *Corpus) Feature(index int) string {
	for k, v := range cp.features {
		if v == index {
			return k
		}
	}
	return ""
}

//Classifier 分类器
type Classifier func(string) string

//LibSvmClassifier 基于libsvm的分类器
func LibSvmClassifier(labelFile, featureFile, trainFile, modelFile string) (Classifier, error) {
	tf, err := os.Open(trainFile)
	if err != nil {
		return nil, err
	}
	defer tf.Close()

	var (
		classes, features []string
	)
	utils.ProceedLine(labelFile, nil, func(line []byte) {
		items := bytes.Split(line, []byte(":"))
		if len(items) == 2 {
			classes = append(classes, string(bytes.TrimSpace(items[0])))
		}
	})
	utils.ProceedLine(featureFile, nil, func(line []byte) {
		feature := bytes.TrimSpace(line)
		if len(feature) != 0 {
			features = append(features, string(feature))
		}
	})
	corpus := NewCorpus(classes, features, LoadTraning(tf))
	model := libSvm.NewModelFromFile(modelFile)

	return func(txt string) string {
		vector := corpus.Vector([]byte(txt))
		label := model.Predict(vector)
		return corpus.Label(int(label))
	}, nil
}

//LibSvmFromZip 加载zip语料的svm分类器
func LibSvmFromZip(fpath string) (Classifier, error) {
	mz, err := utils.OpenMemZip(fpath)
	if err != nil {
		return nil, err
	}
	mr, err := mz.Get("train.date.model")
	if err != nil {
		return nil, err
	}
	corpus, err := NewZipCorpus(mz)
	if err != nil {
		return nil, err
	}

	model := libSvm.NewModel(libSvm.NewParameter())
	model.LoadModel(bufio.NewReader(mr))

	return func(txt string) string {
		vector := corpus.Vector([]byte(txt))
		label := model.Predict(vector)
		return corpus.Label(int(label))
	}, nil
}
